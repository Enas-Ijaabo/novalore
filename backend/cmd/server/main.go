package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/enas/orglens/internal/nova"
	"github.com/enas/orglens/internal/pipeline"
	"github.com/enas/orglens/internal/reader"
)

func main() {
	ctx := context.Background()

	novaClient, err := nova.NewClient(ctx)
	if err != nil {
		log.Fatalf("nova init: %v", err)
	}

	datasetDir := os.Getenv("DATASET_DIR")
	if datasetDir == "" {
		datasetDir = "../dataset"
	}

	chunks, err := reader.ReadDataset(datasetDir)
	if err != nil {
		log.Fatalf("read dataset: %v", err)
	}
	log.Printf("Read %d chunks from dataset", len(chunks))

	var allFacts []pipeline.Fact
	for _, c := range chunks {
		facts, err := novaClient.ExtractFacts(ctx, c.Text, c.Source)
		if err != nil {
			log.Printf("extract [%s]: %v", c.Source, err)
			continue
		}
		allFacts = append(allFacts, facts...)
	}

	log.Printf("Extracted %d facts total", len(allFacts))
	for _, f := range allFacts {
		log.Printf("  %s | %s | %s  [%s]", f.Subject, f.Relation, f.Object, f.Source)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	log.Printf("OrgLens backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
