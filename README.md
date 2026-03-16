# NovaLore

**The knowledge hidden in your code, instantly queryable.**

Drop in any codebase or doc folder. NovaLore uses Amazon Nova to extract factual knowledge statements from every file, stores them as embeddings, and lets you ask natural-language questions — answered with grounded citations pointing back to the source.

No manual documentation. No hallucinations. Just your code, made searchable.

Built for the **Amazon Nova AI Hackathon 2026**.

---

## How it works

```
Drop in your codebase / docs
        │
        ▼
  Nova Lite reads each file and extracts factual knowledge statements
  "Drone assignment uses MySQL spatial indexing to find the nearest idle drone"
        │
        ▼
  Nova Multimodal Embeddings converts each fact to a 1024-dim vector
        │
        ▼
  ChromaDB stores facts + embeddings (persistent volume)
        │
        ▼
  Ask a question → vector search → Nova Lite synthesizes a grounded answer
```

Ingestion starts automatically on startup. The **Ingest** tab shows live per-file progress (`extracting → indexing → done`). Hit **Re-analyze** any time to re-index.

---

## Prerequisites

- Docker & Docker Compose
- AWS credentials with Bedrock model access in **us-east-1**
  - `amazon.nova-lite-v1:0` — fact extraction + answer synthesis
  - `amazon.nova-2-multimodal-embeddings-v1:0` — embeddings

---

## Quickstart

```bash
# 1. Clone
git clone https://github.com/enas/orgLens
cd orgLens

# 2. Set up credentials
cp .env.example .env
# Fill in AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY in .env

# 3. Start everything
docker compose up --build

# 4. Open http://localhost:3000
```

The Ingest tab auto-starts indexing on startup. Switch to **Knowledge** to browse extracted facts, or **Ask** to query in natural language.

---

## Architecture

| Component | Tech |
|---|---|
| Backend API | Go, net/http |
| Fact extraction + synthesis | Amazon Nova Lite (`us.amazon.nova-lite-v1:0`) |
| Embeddings | Amazon Nova Multimodal Embeddings (1024-dim) |
| Vector store | ChromaDB 0.4.24 |
| Frontend | Next.js + Tailwind CSS |
| Orchestration | Docker Compose |

---

## API

| Method | Path | Description |
|---|---|---|
| `GET`  | `/api/ingest/status` | Per-file status `{running, total, files[]}` |
| `POST` | `/api/ingest` | Trigger re-analysis → `202` |
| `GET`  | `/api/facts` | All extracted knowledge statements |
| `POST` | `/api/query` | `{"q": "..."}` → `{answer, sources}` |
| `GET`  | `/api/health` | Health check |

---

## Dataset

The bundled demo dataset is a drone delivery management system — a real Go backend with routes, models, use cases, and infrastructure code.

```
dataset/
  repos/
    drone-delivery-management/
```

To use your own codebase: replace the contents of `dataset/` and hit **Re-analyze**.

---

## Development (without Docker)

```bash
# Terminal 1 — ChromaDB
docker run -p 8001:8000 chromadb/chroma:0.4.24

# Terminal 2 — Backend
cd backend
export $(grep -v '^#' ../.env | xargs) && go run ./cmd/server

# Terminal 3 — Frontend
cd frontend
npm install && npm run dev
```
