package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/enas/novalore/internal/nova"
	"github.com/enas/novalore/internal/pipeline"
	"github.com/enas/novalore/internal/store"
)

// fileStatus tracks the ingestion state of a single dataset file.
type fileStatus struct {
	File      string    `json:"file"`
	Status    string    `json:"status"` // pending, extracting, indexing, done, error
	Facts     int       `json:"facts,omitempty"`
	Error     string    `json:"error,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// ingestJob holds the in-memory state of an ongoing or completed ingest run.
type ingestJob struct {
	mu      sync.RWMutex
	ordered []*fileStatus
	byFile  map[string]*fileStatus
	total   int  // total files in dataset (known upfront)
	running bool
}

func newIngestJob(total int) *ingestJob {
	return &ingestJob{
		total:  total,
		byFile: make(map[string]*fileStatus, total),
	}
}

// add registers a file in the status queue (called just before processing starts).
func (j *ingestJob) add(file string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	s := &fileStatus{File: file, Status: "pending", UpdatedAt: time.Now().UTC()}
	j.ordered = append(j.ordered, s)
	j.byFile[file] = s
}

func (j *ingestJob) set(file, status string, facts int, errMsg string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if s, ok := j.byFile[file]; ok {
		s.Status = status
		s.Facts = facts
		s.Error = errMsg
		s.UpdatedAt = time.Now().UTC()
	}
}

func (j *ingestJob) snapshot() (bool, int, []fileStatus) {
	j.mu.RLock()
	defer j.mu.RUnlock()
	out := make([]fileStatus, len(j.ordered))
	for i, s := range j.ordered {
		out[i] = *s
	}
	return j.running, j.total, out
}

func (j *ingestJob) isRunning() bool {
	j.mu.RLock()
	defer j.mu.RUnlock()
	return j.running
}

type server struct {
	nova       *nova.Client
	chroma     *store.Client
	datasetDir string
	jobMu      sync.Mutex // prevents concurrent ingests
	job        *ingestJob
}

func main() {
	ctx := context.Background()

	novaClient, err := nova.NewClient(ctx)
	if err != nil {
		log.Fatalf("nova init: %v", err)
	}

	chromaClient := store.NewClient()
	if err := chromaClient.EnsureCollection(ctx); err != nil {
		log.Fatalf("chroma init: %v", err)
	}
	if err := chromaClient.EnsureMetaCollection(ctx); err != nil {
		log.Fatalf("chroma meta init: %v", err)
	}

	datasetDir := os.Getenv("DATASET_DIR")
	if datasetDir == "" {
		datasetDir = "../dataset"
	}

	srv := &server{
		nova:       novaClient,
		chroma:     chromaClient,
		datasetDir: datasetDir,
	}

	// Initialise job state from file list so /api/ingest/status is available immediately
	srv.initJob()

	// Auto-start ingestion on startup
	go srv.runIngest(context.Background())

	http.HandleFunc("/api/health", cors(srv.health))
	http.HandleFunc("/api/ingest", cors(srv.triggerIngest))
	http.HandleFunc("/api/ingest/status", cors(srv.ingestStatus))
	http.HandleFunc("/api/facts", cors(srv.facts))
	http.HandleFunc("/api/query", cors(srv.query))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("NovaLore backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// initJob builds the initial job state from the dataset file list.
func (s *server) initJob() {
	files, err := pipeline.WalkDataset(s.datasetDir)
	if err != nil {
		log.Printf("initJob walk: %v", err)
		s.job = newIngestJob(0)
		return
	}
	s.job = newIngestJob(len(files))
}

// runIngest processes all dataset files sequentially: extract → embed → store.
// Writes per-file status updates to in-memory job as it goes.
func (s *server) runIngest(ctx context.Context) {
	s.jobMu.Lock()
	if s.job.isRunning() {
		s.jobMu.Unlock()
		log.Println("[ingest] already running, skipping")
		return
	}
	// Reset job (empty — files added one by one as processing begins)
	s.initJob()
	s.job.mu.Lock()
	s.job.running = true
	s.job.mu.Unlock()
	s.jobMu.Unlock()

	defer func() {
		s.job.mu.Lock()
		s.job.running = false
		s.job.mu.Unlock()
		log.Println("[ingest] run complete")
	}()

	log.Println("[ingest] starting")

	files, err := pipeline.WalkDataset(s.datasetDir)
	if err != nil {
		log.Printf("[ingest] walk: %v", err)
		return
	}

	seen := map[string]bool{} // cross-file dedup
	total := 0

	for i, absPath := range files {
		relPath, _ := filepath.Rel(s.datasetDir, absPath)

		// Register file in status queue just before processing
		s.job.add(relPath)

		// Wipe existing facts for this file before re-extracting
		if err := s.chroma.DeleteBySource(ctx, relPath); err != nil {
			log.Printf("[ingest] delete source %s: %v", relPath, err)
		}
		if err := s.chroma.DeleteFileMeta(ctx, relPath); err != nil {
			log.Printf("[ingest] delete meta %s: %v", relPath, err)
		}

		// --- Phase 1: Extract ---
		log.Printf("[ingest] extracting: %s (%d/%d)", relPath, i+1, len(files))
		s.job.set(relPath, "extracting", 0, "")

		chunks, err := pipeline.ChunkFile(absPath)
		if err != nil {
			log.Printf("[ingest] chunk %s: %v", relPath, err)
			s.job.set(relPath, "error", 0, err.Error())
			continue
		}
		for i := range chunks {
			chunks[i].Source = relPath
		}

		var rawFacts []pipeline.Fact
		for _, chunk := range chunks {
			ff, err := s.nova.ExtractFacts(ctx, chunk.Content, chunk.Source)
			if err != nil {
				log.Printf("[ingest] extract %s: %v", relPath, err)
				continue
			}
			rawFacts = append(rawFacts, ff...)
		}

		// Deduplicate: within-file + cross-file
		var facts []pipeline.Fact
		for _, f := range rawFacts {
			if key := pipeline.NormalizeText(f.Text); !seen[key] {
				seen[key] = true
				facts = append(facts, f)
			}
		}

		if len(facts) == 0 {
			log.Printf("[ingest] no new facts in %s", relPath)
			s.job.set(relPath, "done", 0, "")
			if err := s.chroma.WriteFileMeta(ctx, relPath, time.Now().UTC(), 0); err != nil {
				log.Printf("[ingest] meta %s: %v", relPath, err)
			}
			continue
		}

		// --- Phase 2: Embed + Store ---
		log.Printf("[ingest] indexing: %s (%d new facts)", relPath, len(facts))
		s.job.set(relPath, "indexing", 0, "")

		embeddings := make([][]float64, len(facts))
		embedErr := false
		for i, f := range facts {
			emb, err := s.nova.Embed(ctx, f.Text)
			if err != nil {
				log.Printf("[ingest] embed %s[%d]: %v", relPath, i, err)
				s.job.set(relPath, "error", 0, err.Error())
				embedErr = true
				break
			}
			embeddings[i] = emb
			time.Sleep(300 * time.Millisecond)
		}
		if embedErr {
			continue
		}

		if err := s.chroma.Add(ctx, facts, embeddings); err != nil {
			log.Printf("[ingest] store %s: %v", relPath, err)
			s.job.set(relPath, "error", 0, err.Error())
			continue
		}

		if err := s.chroma.WriteFileMeta(ctx, relPath, time.Now().UTC(), len(facts)); err != nil {
			log.Printf("[ingest] meta %s: %v", relPath, err)
		}

		total += len(facts)
		log.Printf("[ingest] done: %s (%d facts, %d total)", relPath, len(facts), total)
		s.job.set(relPath, "done", len(facts), "")

		// Brief pause between files to stay under Bedrock rate limits
		time.Sleep(1 * time.Second)
	}

	log.Printf("[ingest] finished — %d facts indexed across %d files", total, len(files))
}

func cors(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}

func (s *server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// triggerIngest starts a re-ingest in the background and returns 202 immediately.
func (s *server) triggerIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.job.isRunning() {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"status": "already_running"})
		return
	}
	go s.runIngest(context.Background())
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"status": "started"})
}

// ingestStatus returns the per-file ingestion status from in-memory job state.
func (s *server) ingestStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.job == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"running": false, "total": 0, "files": []fileStatus{}})
		return
	}
	running, total, files := s.job.snapshot()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"running": running, "total": total, "files": files})
}

func (s *server) facts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	facts, err := s.chroma.GetAll(r.Context())
	if err != nil {
		log.Printf("getall: %v", err)
		http.Error(w, "store read failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(facts)
}

func (s *server) query(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	var req struct {
		Q string `json:"q"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Q == "" {
		http.Error(w, `body must be {"q": "..."}`, http.StatusBadRequest)
		return
	}

	emb, err := s.nova.Embed(ctx, req.Q)
	if err != nil {
		log.Printf("query embed: %v", err)
		http.Error(w, "embed failed", http.StatusInternalServerError)
		return
	}

	facts, err := s.chroma.Query(ctx, emb, 10)
	if err != nil {
		log.Printf("query search: %v", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	answer, err := s.nova.Synthesize(ctx, req.Q, facts)
	if err != nil {
		log.Printf("query synthesize: %v", err)
		http.Error(w, "synthesis failed", http.StatusInternalServerError)
		return
	}

	type source struct {
		Text   string `json:"text"`
		Source string `json:"source"`
	}
	sources := make([]source, len(facts))
	for i, f := range facts {
		sources[i] = source{Text: f.Text, Source: f.Source}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"answer":  answer,
		"sources": sources,
	})
}
