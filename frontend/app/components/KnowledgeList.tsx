'use client'

interface Fact {
  ID: string
  Text: string
  Type: string
  Source: string
}

interface Props {
  facts: Fact[]
}

const TYPE_META: Record<string, { label: string; color: string }> = {
  business_rule: { label: 'Business Rules',   color: 'bg-violet-500/10 text-violet-300 border-violet-500/20' },
  architecture:  { label: 'Architecture',      color: 'bg-blue-500/10 text-blue-300 border-blue-500/20' },
  constraint:    { label: 'Constraints',       color: 'bg-amber-500/10 text-amber-300 border-amber-500/20' },
  behavior:      { label: 'Behaviors',         color: 'bg-emerald-500/10 text-emerald-300 border-emerald-500/20' },
  data_rule:     { label: 'Data Rules',        color: 'bg-teal-500/10 text-teal-300 border-teal-500/20' },
  decision:      { label: 'Decisions',         color: 'bg-rose-500/10 text-rose-300 border-rose-500/20' },
}

const TYPE_ORDER = ['business_rule', 'architecture', 'constraint', 'behavior', 'data_rule', 'decision']

export default function KnowledgeList({ facts }: Props) {
  if (facts.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-48 text-zinc-600 text-sm">
        Run "Analyze Codebase" to populate the knowledge base.
      </div>
    )
  }

  const grouped = TYPE_ORDER.reduce<Record<string, Fact[]>>((acc, type) => {
    const group = facts.filter((f) => f.Type === type)
    if (group.length > 0) acc[type] = group
    return acc
  }, {})

  return (
    <div className="flex flex-col gap-6">
      {Object.entries(grouped).map(([type, group]) => {
        const meta = TYPE_META[type] ?? { label: type, color: 'bg-zinc-500/10 text-zinc-300 border-zinc-500/20' }
        return (
          <section key={type}>
            <h3 className="text-xs font-semibold uppercase tracking-widest text-zinc-400 mb-2">
              {meta.label}
              <span className="ml-2 text-zinc-600 normal-case font-normal">({group.length})</span>
            </h3>
            <ul className="flex flex-col gap-1.5">
              {group.map((f) => (
                <li
                  key={f.ID}
                  className="flex items-start justify-between gap-3 rounded-lg border border-zinc-800 bg-zinc-900 px-3 py-2 text-sm"
                >
                  <span className="text-zinc-200 leading-relaxed">{f.Text}</span>
                  <span className={`shrink-0 rounded border px-1.5 py-0.5 text-xs font-mono ${meta.color}`}>
                    {f.Source.split('/').pop()}
                  </span>
                </li>
              ))}
            </ul>
          </section>
        )
      })}
    </div>
  )
}
