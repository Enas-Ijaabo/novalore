'use client'

import { useState, useEffect, useRef } from 'react'
import { IngestStatus, FileStatus } from '../lib/api'

const API = process.env.NEXT_PUBLIC_API_URL ?? 'http://localhost:8080'

interface Props {
  initialData: IngestStatus
}

function timeAgo(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  return `${Math.floor(hours / 24)}d ago`
}

function groupByFolder(files: FileStatus[]): [string, FileStatus[]][] {
  const map = new Map<string, FileStatus[]>()
  for (const f of files) {
    const parts = f.file.split('/')
    const key = parts.length > 1 ? parts.slice(0, -1).join('/') + '/' : './'
    if (!map.has(key)) map.set(key, [])
    map.get(key)!.push(f)
  }
  return Array.from(map.entries())
}

const STATUS_ICON: Record<string, { icon: string; cls: string; spin?: boolean }> = {
  pending:    { icon: '—',  cls: 'text-zinc-600' },
  extracting: { icon: '⟳', cls: 'text-yellow-400', spin: true },
  indexing:   { icon: '⟳', cls: 'text-blue-400',   spin: true },
  done:       { icon: '✓',  cls: 'text-emerald-400' },
  error:      { icon: '✗',  cls: 'text-red-400' },
}

const STATUS_LABEL: Record<string, string> = {
  pending:    'waiting',
  extracting: 'extracting…',
  indexing:   'indexing…',
  done:       '',
  error:      'error',
}

export default function IngestTab({ initialData }: Props) {
  const [data, setData] = useState<IngestStatus>(initialData)
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const { running, total, files } = data
  const done      = files.filter(f => f.status === 'done').length
  const errCount  = files.filter(f => f.status === 'error').length
  const totalFacts = files.reduce((s, f) => s + (f.facts ?? 0), 0)
  const lastDone  = files.filter(f => f.status === 'done' && f.updated_at).map(f => new Date(f.updated_at!).getTime())
  const lastAnalysis = lastDone.length ? timeAgo(new Date(Math.max(...lastDone)).toISOString()) : 'never'
  const allDone = total > 0 && done === total && !running

  const fetchStatus = async () => {
    try {
      const res = await fetch(`${API}/api/ingest/status`)
      if (!res.ok) return
      const next: IngestStatus = await res.json()
      setData(next)
      if (!next.running && pollRef.current) {
        clearInterval(pollRef.current)
        pollRef.current = null
      }
    } catch {}
  }

  const startPolling = () => {
    if (pollRef.current) return
    pollRef.current = setInterval(fetchStatus, 2000)
  }

  useEffect(() => {
    if (initialData.running) startPolling()
    return () => { if (pollRef.current) clearInterval(pollRef.current) }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const handleReanalyze = async () => {
    try {
      const res = await fetch(`${API}/api/ingest`, { method: 'POST' })
      if (res.ok || res.status === 202) {
        await fetchStatus()
        startPolling()
      }
    } catch {}
  }

  const folders = groupByFolder(files)

  return (
    <div className="h-full overflow-y-auto">
      <div className="max-w-3xl mx-auto px-6 py-8 space-y-6">

        {/* Stats */}
        <div className="grid grid-cols-3 gap-4">
          {[
            { label: 'Files indexed',    value: total ? `${done} / ${total}` : '—' },
            { label: 'Facts discovered', value: totalFacts || (running ? '…' : '—') },
            { label: 'Last analysis',    value: lastAnalysis },
          ].map(stat => (
            <div key={stat.label} className="bg-zinc-900 border border-zinc-800 rounded-xl px-5 py-4">
              <div className="text-xs text-zinc-500 mb-1.5">{stat.label}</div>
              <div className="text-xl font-semibold text-white">{stat.value}</div>
            </div>
          ))}
        </div>

        {/* File list */}
        <div className="bg-zinc-900 border border-zinc-800 rounded-xl overflow-hidden">
          <div className="flex items-center justify-between px-5 py-4 border-b border-zinc-800">
            <div className="flex items-center gap-3">
              <span className="text-sm font-medium text-zinc-300">Dataset files</span>
              {running && (
                <span className="flex items-center gap-1.5 text-xs text-yellow-400">
                  <span className="w-1.5 h-1.5 bg-yellow-400 rounded-full animate-pulse" />
                  Indexing
                </span>
              )}
              {allDone && !errCount && (
                <span className="flex items-center gap-1.5 text-xs text-emerald-400">
                  <span className="w-1.5 h-1.5 bg-emerald-400 rounded-full" />
                  Complete
                </span>
              )}
              {errCount > 0 && (
                <span className="text-xs text-red-400">{errCount} error{errCount > 1 ? 's' : ''}</span>
              )}
            </div>
            {!running && files.length > 0 && (
              <button onClick={handleReanalyze}
                className="text-xs px-3 py-1.5 rounded-lg border border-zinc-700 text-zinc-400 hover:text-zinc-200 hover:border-zinc-600 transition-colors">
                Re-analyze
              </button>
            )}
          </div>

          <div className="px-5 py-4">
            {files.length === 0 ? (
              <div className="flex items-center gap-3 py-10 justify-center">
                <span className="w-3 h-3 border border-zinc-600 border-t-zinc-300 rounded-full animate-spin" />
                <span className="text-zinc-500 text-sm">Starting indexer…</span>
              </div>
            ) : (
              <div className="space-y-5">
                {folders.map(([folder, folderFiles]) => (
                  <div key={folder}>
                    <div className="text-[10px] font-semibold text-zinc-600 uppercase tracking-widest mb-2">{folder}</div>
                    <div className="space-y-0.5">
                      {folderFiles.map(f => {
                        const s = STATUS_ICON[f.status] ?? STATUS_ICON.pending
                        return (
                          <div key={f.file} className="flex items-center gap-3 px-2 py-1.5 rounded-lg hover:bg-zinc-800/50 transition-colors">
                            <span className={`w-3.5 text-[11px] shrink-0 ${s.cls} ${s.spin ? 'animate-spin' : ''}`}>{s.icon}</span>
                            <span className={`font-mono text-xs flex-1 truncate ${
                              f.status === 'done' ? 'text-zinc-300'
                              : f.status === 'error' ? 'text-red-400'
                              : f.status === 'extracting' || f.status === 'indexing' ? 'text-zinc-200'
                              : 'text-zinc-600'
                            }`}>
                              {f.file.split('/').pop()}
                            </span>
                            <span className="text-zinc-600 text-xs shrink-0 w-24 text-right">
                              {f.status === 'done' && f.updated_at ? timeAgo(f.updated_at) : STATUS_LABEL[f.status]}
                            </span>
                            {f.status === 'done' && f.facts !== undefined && (
                              <span className="text-zinc-600 text-xs font-mono whitespace-nowrap shrink-0">[{f.facts} facts]</span>
                            )}
                            {f.status === 'error' && f.error && (
                              <span className="text-red-600 text-xs truncate max-w-[120px]" title={f.error}>{f.error}</span>
                            )}
                          </div>
                        )
                      })}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

      </div>
    </div>
  )
}
