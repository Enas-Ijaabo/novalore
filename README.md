# NovaLore

**Codebase knowledge, instantly queryable.**

NovaLore ingests your source code and docs, uses Amazon Nova to extract factual knowledge statements, stores them as embeddings in ChromaDB, and lets you ask natural-language questions answered with grounded citations.

Built for the **Amazon Nova AI Hackathon 2026**.

---

## How it works

```
Backend starts
        │
        ▼
  Background goroutine walks dataset files one by one
  For each file: extract → embed → store
  Status written to ChromaDB metadata as it goes
        │
        ▼
  Nova Lite extracts knowledge statements per file
  "JWT tokens expire after 24 hours [auth.go]"
        │
        ▼
  Nova Embeddings (1024-dim vectors, sequential, rate-limit-aware)
        │
        ▼
  ChromaDB vector store (persistent volume)
        │
        ▼
  Query → vector search → Nova Lite synthesis → grounded answer
```

Ingestion runs automatically on startup — no button click needed. The **Ingest** tab is a live monitor showing each file's status (`pending → extracting → indexing → done`). Use the **Re-analyze** button to re-trigger at any time.

---

## Prerequisites

- Docker & Docker Compose
- AWS credentials with Bedrock model access enabled in **us-east-1**
  - `amazon.nova-lite-v1:0` — fact extraction + synthesis
  - `amazon.nova-2-multimodal-embeddings-v1:0` — embeddings

---

## Quickstart

```bash
# 1. Clone
git clone https://github.com/enas/novalore
cd novalore

# 2. Create .env from the example and fill in your AWS credentials
cp .env.example .env
# edit .env

# 3. Start everything
docker compose up --build

# 4. Open http://localhost:3000
#    The Ingest tab shows live indexing progress (auto-starts on startup)
#    Switch to Knowledge to browse extracted facts
#    Switch to Ask to query the codebase in natural language
```

---

## Architecture

| Component | Tech |
|---|---|
| Backend API | Go, net/http |
| LLM extraction + synthesis | Amazon Nova Lite (Bedrock) |
| Embeddings | Amazon Nova Multimodal Embeddings (1024-dim) |
| Vector store | ChromaDB 0.4.24 |
| Frontend | Next.js 16 + Tailwind CSS |
| Container orchestration | Docker Compose |

---

## API

| Method | Path | Description |
|---|---|---|
| `GET`  | `/api/ingest/status` | Per-file status `{running, total, files[{file, status, facts, updated_at}]}` |
| `POST` | `/api/ingest` | Trigger re-analysis in background → `202 {status: "started"}` |
| `GET`  | `/api/facts` | All stored knowledge statements |
| `POST` | `/api/query` | `{"q": "..."}` → `{answer, sources}` |
| `GET`  | `/api/health` | Health check |

File status values: `pending` → `extracting` → `indexing` → `done` / `error`

---

## Dataset

The bundled dataset lives in `dataset/` and includes simulated services and docs:

```
dataset/
  docs/
    architecture_overview.txt
    auth_design_doc.txt
    meeting_notes.txt
  repos/
    auth-service/
    payment-service/
    api-gateway/
```

Swap in your own codebase by replacing the contents of `dataset/` before running ingest.

---

## Development (without Docker)

```bash
# Terminal 1 — ChromaDB
docker run -p 8001:8000 chromadb/chroma:0.4.24

# Terminal 2 — Backend (reads .env automatically via direnv, or set vars inline)
cd backend
export $(grep -v '^#' ../.env | xargs) && go run ./cmd/server

# Terminal 3 — Frontend
cd frontend
npm install
npm run dev
```
