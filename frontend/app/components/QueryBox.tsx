'use client'

import { useState } from 'react'

interface Source {
  text: string
  source: string
}

interface Props {
  disabled: boolean
}

const API = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

export default function QueryBox({ disabled }: Props) {
  const [question, setQuestion] = useState('')
  const [loading, setLoading] = useState(false)
  const [answer, setAnswer] = useState<string | null>(null)
  const [sources, setSources] = useState<Source[]>([])
  const [error, setError] = useState<string | null>(null)

  async function submit(e: React.FormEvent) {
    e.preventDefault()
    if (!question.trim() || loading) return
    setLoading(true)
    setAnswer(null)
    setSources([])
    setError(null)

    try {
      const res = await fetch(`${API}/api/query`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ q: question }),
      })
      if (!res.ok) throw new Error(await res.text())
      const data = await res.json()
      setAnswer(data.answer)
      setSources(data.sources ?? [])
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Query failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex flex-col gap-4">
      <h2 className="text-xs font-semibold uppercase tracking-widest text-zinc-400">
        Query
      </h2>

      <form onSubmit={submit} className="flex gap-2">
        <input
          type="text"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          placeholder="How does authentication work?"
          disabled={disabled || loading}
          className="flex-1 rounded-lg border border-zinc-700 bg-zinc-900 px-4 py-2.5 text-sm text-zinc-100 placeholder:text-zinc-600 focus:border-indigo-500 focus:outline-none disabled:opacity-40"
        />
        <button
          type="submit"
          disabled={disabled || loading || !question.trim()}
          className="rounded-lg bg-indigo-600 px-5 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-500 disabled:opacity-40 disabled:cursor-not-allowed"
        >
          {loading ? (
            <span className="inline-block h-4 w-4 rounded-full border-2 border-white border-t-transparent animate-spin" />
          ) : (
            'Ask'
          )}
        </button>
      </form>

      {error && (
        <p className="text-sm text-red-400">{error}</p>
      )}

      {answer && (
        <div className="flex flex-col gap-3 rounded-lg border border-zinc-800 bg-zinc-900 p-4">
          <p className="text-sm text-zinc-200 leading-relaxed whitespace-pre-wrap">{answer}</p>
          {sources.length > 0 && (
            <div className="border-t border-zinc-800 pt-3">
              <p className="text-xs font-semibold uppercase tracking-widest text-zinc-500 mb-2">Sources</p>
              <ul className="flex flex-wrap gap-2">
                {sources.map((s, i) => (
                  <li
                    key={i}
                    title={s.text}
                    className="rounded border border-zinc-700 bg-zinc-800 px-2 py-1 text-xs font-mono text-zinc-400"
                  >
                    {s.source.split('/').pop()}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>
      )}
    </div>
  )
}
