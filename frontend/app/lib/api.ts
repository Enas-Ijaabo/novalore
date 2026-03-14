// Server-side API helpers — uses INTERNAL_API_URL to reach backend within Docker network.
// Falls back to localhost for local development.

const BASE = process.env.INTERNAL_API_URL ?? 'http://localhost:8080'

export interface Fact {
  ID: string
  Text: string
  Type: string
  Source: string
}

export type FileStatusState = 'pending' | 'extracting' | 'indexing' | 'done' | 'error'

export interface FileStatus {
  file: string
  status: FileStatusState
  facts?: number
  error?: string
  updated_at?: string
}

export interface IngestStatus {
  running: boolean
  total: number
  files: FileStatus[]
}

export async function getFacts(): Promise<Fact[]> {
  try {
    const res = await fetch(`${BASE}/api/facts`, { cache: 'no-store' })
    if (!res.ok) return []
    const data = await res.json()
    return data ?? []
  } catch {
    return []
  }
}

export async function getIngestStatus(): Promise<IngestStatus> {
  try {
    const res = await fetch(`${BASE}/api/ingest/status`, { cache: 'no-store' })
    if (!res.ok) return { running: false, total: 0, files: [] }
    const data = await res.json()
    return data ?? { running: false, total: 0, files: [] }
  } catch {
    return { running: false, total: 0, files: [] }
  }
}
