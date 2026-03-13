'use client'

import { useState } from 'react'
import SourcePanel from './components/SourcePanel'
import KnowledgeList from './components/KnowledgeList'
import QueryBox from './components/QueryBox'

interface Fact {
  ID: string
  Text: string
  Type: string
  Source: string
}

const API = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

export default function Home() {
  const [facts, setFacts] = useState<Fact[]>([])
  const [analyzing, setAnalyzing] = useState(false)
  const [factsCount, setFactsCount] = useState<number | null>(null)

  async function handleAnalyze() {
    setAnalyzing(true)
    try {
      await fetch(`${API}/api/ingest`, { method: 'POST' })
      const res = await fetch(`${API}/api/facts`)
      const data: Fact[] = await res.json()
      setFacts(data)
      setFactsCount(data.length)
    } finally {
      setAnalyzing(false)
    }
  }

  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-100 font-sans">
      {/* Header */}
      <header className="border-b border-zinc-800 px-8 py-4 flex items-center gap-3">
        <span className="text-lg font-bold tracking-tight">OrgLens</span>
        <span className="text-zinc-600 text-sm">|</span>
        <span className="text-sm text-zinc-400">Codebase knowledge, instantly queryable</span>
      </header>

      {/* Body */}
      <div className="flex gap-8 px-8 py-8 max-w-screen-xl mx-auto">
        {/* Left: SourcePanel */}
        <SourcePanel
          analyzing={analyzing}
          factsCount={factsCount}
          onAnalyze={handleAnalyze}
        />

        {/* Right: KnowledgeList + QueryBox */}
        <main className="flex-1 flex flex-col gap-8 min-w-0">
          <section>
            <h2 className="text-xs font-semibold uppercase tracking-widest text-zinc-400 mb-4">
              Knowledge Base
              {facts.length > 0 && (
                <span className="ml-2 text-zinc-600 normal-case font-normal">
                  ({facts.length} statements)
                </span>
              )}
            </h2>
            <KnowledgeList facts={facts} />
          </section>

          <div className="border-t border-zinc-800 pt-8">
            <QueryBox disabled={facts.length === 0} />
          </div>
        </main>
      </div>
    </div>
  )
}
