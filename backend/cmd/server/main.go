package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/enas/orglens/internal/nova"
	"github.com/enas/orglens/internal/pipeline"
	"github.com/enas/orglens/internal/store"
)

type server struct {
	nova       *nova.Client
	chroma     *store.Client
	datasetDir string
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

	datasetDir := os.Getenv("DATASET_DIR")
	if datasetDir == "" {
		datasetDir = "../dataset"
	}

	srv := &server{
		nova:       novaClient,
		chroma:     chromaClient,
		datasetDir: datasetDir,
	}

	http.HandleFunc("/api/health", cors(srv.health))
	http.HandleFunc("/api/ingest", cors(srv.ingest))
	http.HandleFunc("/api/facts", cors(srv.facts))
	http.HandleFunc("/api/query", cors(srv.query))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("OrgLens backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
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

func (s *server) ingest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	// 1. Extract facts from dataset
	facts, err := pipeline.Run(ctx, s.datasetDir, s.nova.ExtractFacts)
	if err != nil {
		log.Printf("pipeline error: %v", err)
		http.Error(w, "pipeline failed", http.StatusInternalServerError)
		return
	}

	// 2. Fresh ingest — reset collection
	if err := s.chroma.Reset(ctx); err != nil {
		log.Printf("chroma reset: %v", err)
		http.Error(w, "store reset failed", http.StatusInternalServerError)
		return
	}

	// 3. Embed each fact
	log.Printf("Embedding %d facts...", len(facts))
	embeddings := make([][]float64, len(facts))
	for i, f := range facts {
		emb, err := s.nova.Embed(ctx, f.Text)
		if err != nil {
			log.Printf("embed [%d]: %v", i, err)
			http.Error(w, "embedding failed", http.StatusInternalServerError)
			return
		}
		embeddings[i] = emb
	}

	// 4. Store in ChromaDB
	if err := s.chroma.Add(ctx, facts, embeddings); err != nil {
		log.Printf("chroma add: %v", err)
		http.Error(w, "store failed", http.StatusInternalServerError)
		return
	}

	log.Printf("Stored %d facts in ChromaDB", len(facts))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"facts_count": len(facts)})
}

func (s *server) facts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ctx := r.Context()

	facts, err := s.chroma.GetAll(ctx)
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
		http.Error(w, "body must be {\"q\": \"...\"}", http.StatusBadRequest)
		return
	}

	// 1. Embed the question
	emb, err := s.nova.Embed(ctx, req.Q)
	if err != nil {
		log.Printf("query embed: %v", err)
		http.Error(w, "embed failed", http.StatusInternalServerError)
		return
	}

	// 2. Vector search — top 10 facts
	facts, err := s.chroma.Query(ctx, emb, 10)
	if err != nil {
		log.Printf("query search: %v", err)
		http.Error(w, "search failed", http.StatusInternalServerError)
		return
	}

	// 3. Nova synthesis
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
