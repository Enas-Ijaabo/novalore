'use client'

const SOURCES = [
  {
    label: 'repos/auth-service',
    files: ['auth.go', 'config.yaml', 'README.md'],
  },
  {
    label: 'repos/payment-service',
    files: ['payment.go', 'reconciliation.go', 'stripe.go', 'config.yaml', 'README.md'],
  },
  {
    label: 'repos/api-gateway',
    files: ['gateway.go', 'routes.go', 'README.md'],
  },
  {
    label: 'docs',
    files: ['architecture_overview.txt', 'auth_design.txt', 'meeting_notes.txt'],
  },
]

interface Props {
  analyzing: boolean
  factsCount: number | null
  onAnalyze: () => void
}

export default function SourcePanel({ analyzing, factsCount, onAnalyze }: Props) {
  return (
    <aside className="flex flex-col gap-6 w-64 shrink-0">
      <div>
        <h2 className="text-xs font-semibold uppercase tracking-widest text-zinc-400 mb-3">
          Dataset
        </h2>
        <ul className="flex flex-col gap-4">
          {SOURCES.map((src) => (
            <li key={src.label}>
              <p className="text-sm font-medium text-zinc-300 mb-1">{src.label}</p>
              <ul className="flex flex-col gap-0.5 pl-3 border-l border-zinc-700">
                {src.files.map((f) => (
                  <li key={f} className="text-xs text-zinc-500">
                    {f}
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
      </div>

      <button
        onClick={onAnalyze}
        disabled={analyzing}
        className="flex items-center justify-center gap-2 rounded-lg bg-indigo-600 px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {analyzing ? (
          <>
            <span className="inline-block h-3.5 w-3.5 rounded-full border-2 border-white border-t-transparent animate-spin" />
            Analyzing…
          </>
        ) : (
          'Analyze Codebase'
        )}
      </button>

      {factsCount !== null && !analyzing && (
        <p className="text-xs text-emerald-400 font-medium">
          ✓ {factsCount} knowledge statements stored
        </p>
      )}
    </aside>
  )
}
