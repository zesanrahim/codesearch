import { useState } from 'react'
import { diff, tokens } from './demoData'

function Usage({ u }) {
  return (
    <div className="px-4 py-2.5 border-b border-line1/60 hover:bg-ink2 transition-colors">
      <div className="font-mono text-[11.5px] text-dim">
        {u.path}
        <span className="text-faint">:{u.line}</span>
        {u.self ? <span className="text-faint"> · this diff</span> : null}
      </div>
      <div className="mt-1 font-mono text-[12px] text-cream truncate">
        {u.code}
      </div>
    </div>
  )
}

function Panel({ active }) {
  const tok = active ? tokens[active] : null

  if (!tok) {
    return (
      <aside className="border-t md:border-t-0 md:border-l border-line1 bg-ink1 flex items-center justify-center p-8 text-center text-[13px] text-faint">
        Hover an underlined identifier.
      </aside>
    )
  }

  return (
    <aside className="border-t md:border-t-0 md:border-l border-line1 bg-ink1 flex flex-col min-h-[372px]">
      <div className="px-4 py-3.5 border-b border-line1">
        <div className="font-mono text-[13.5px] text-t5">{tok.name}</div>
        <div className="mt-1 text-[12.5px] text-dim">
          {tok.count === 0 ? (
            <>
              <b className="text-cream font-semibold">No</b> occurrences
              elsewhere
            </>
          ) : (
            <>
              <b className="text-cream font-semibold">{tok.count}</b> usages
              across <b className="text-cream font-semibold">{tok.files}</b>{' '}
              files
            </>
          )}
        </div>
      </div>

      {tok.flag ? (
        <div className="m-4 p-3 rounded-md border border-c3/40 bg-c3/10 text-[12.5px] text-dim">
          <b className="block mb-1 font-mono text-[11px] tracking-[0.1em] uppercase font-medium text-t3">
            {tok.flag.title}
          </b>
          {tok.flag.body}
        </div>
      ) : null}

      <div className="flex-1 overflow-y-auto py-1.5">
        {tok.usages.map((u) => (
          <Usage key={u.path + u.line} u={u} />
        ))}
        {tok.neighbours?.length ? (
          <>
            <div className="px-4 pt-3 pb-1 font-mono text-[11px] tracking-[0.1em] uppercase text-faint">
              similar names nearby
            </div>
            {tok.neighbours.map((u) => (
              <Usage key={u.path + u.line} u={u} />
            ))}
          </>
        ) : null}
      </div>
    </aside>
  )
}

export default function DiffDemo() {
  const [active, setActive] = useState('getSubtotal')

  return (
    <div
      id="demo"
      className="rounded-xl border border-line1 bg-gradient-to-b from-ink1 to-ink overflow-hidden shadow-[0_32px_80px_-32px_rgba(0,0,0,0.9)]"
    >
      <div className="flex items-center gap-3 h-11 px-3.5 border-b border-line1 bg-ink2">
        <div className="flex gap-1.5">
          <i className="w-2.5 h-2.5 rounded-full bg-c1/80" />
          <i className="w-2.5 h-2.5 rounded-full bg-c3/80" />
          <i className="w-2.5 h-2.5 rounded-full bg-c4/80" />
        </div>
        <div className="font-mono text-[12.5px] text-dim">
          internal/billing/invoice.go
        </div>
        <div className="ml-auto flex items-center gap-2.5 font-mono text-[11.5px]">
          <span className="px-2 py-0.5 rounded-full border border-line2 text-dim">
            #412
          </span>
          <span className="text-t4">+8</span>
          <span className="text-t1">−1</span>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-[minmax(0,1fr)_356px]">
        <div className="py-3.5 font-mono text-[13px] leading-[1.85] overflow-x-auto">
          {diff.map((row, i) =>
            row.hunk ? (
              <div
                key={i}
                className="px-4 py-1 mb-2 text-[12px] text-faint bg-cream/[0.02] border-y border-line1"
              >
                {row.hunk}
              </div>
            ) : (
              <div
                key={i}
                className={[
                  'flex gap-3.5 px-4 whitespace-pre border-l-2',
                  row.kind === 'add'
                    ? 'bg-c4/[0.14] border-l-c4'
                    : row.kind === 'del'
                      ? 'bg-c1/[0.12] border-l-c1'
                      : 'border-l-transparent',
                ].join(' ')}
              >
                <span className="w-7 shrink-0 text-right text-faint select-none">
                  {row.ln}
                </span>
                <span
                  className={[
                    'w-2 shrink-0 select-none',
                    row.kind === 'add'
                      ? 'text-t4'
                      : row.kind === 'del'
                        ? 'text-t1'
                        : 'text-faint',
                  ].join(' ')}
                >
                  {row.kind === 'add' ? '+' : row.kind === 'del' ? '−' : ' '}
                </span>
                <span className="min-w-0">
                  {row.seg.map((s, j) => {
                    if (s.k === 'kw') {
                      return (
                        <span className="text-t5" key={j}>
                          {s.v}
                        </span>
                      )
                    }
                    if (s.k === 'tok') {
                      const flagged = Boolean(tokens[s.ref]?.flag)
                      return (
                        <span
                          key={j}
                          tabIndex={0}
                          onMouseEnter={() => setActive(s.ref)}
                          onFocus={() => setActive(s.ref)}
                          className={[
                            'tok',
                            flagged ? 'flagged' : '',
                            active === s.ref ? 'active' : '',
                          ]
                            .filter(Boolean)
                            .join(' ')}
                        >
                          {s.v}
                        </span>
                      )
                    }
                    return <span key={j}>{s.v}</span>
                  })}
                </span>
              </div>
            )
          )}
        </div>

        <Panel active={active} />
      </div>
    </div>
  )
}
