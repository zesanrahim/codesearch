import { useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { fetchPullRequest, useAsync } from '../api'

const KIND_STYLE = {
  add: 'bg-c4/[0.14] border-l-c4',
  del: 'bg-c1/[0.12] border-l-c1',
  ctx: 'border-l-transparent',
}

const SIGN = { add: '+', del: '−', ctx: ' ' }

function Comment({ c }) {
  return (
    <div className="my-1.5 mx-10 p-3 rounded-md border border-line1 bg-ink1">
      <div className="flex items-center gap-2 mb-1.5">
        <img src={c.avatarUrl} alt="" className="w-4 h-4 rounded-full" />
        <span className="font-mono text-[11.5px] text-dim">{c.author}</span>
        <a
          href={c.url}
          target="_blank"
          rel="noreferrer"
          className="ml-auto font-mono text-[11px] text-faint hover:text-cream"
        >
          github ↗
        </a>
      </div>
      <div className="text-[13px] text-cream whitespace-pre-wrap font-sans">
        {c.body}
      </div>
    </div>
  )
}

function Hunk({ hunk, comments }) {
  return (
    <div>
      <div className="px-4 py-1 text-[12px] text-faint bg-cream/[0.02] border-y border-line1 font-mono">
        {hunk.header}
      </div>

      {hunk.lines.map((l, i) => {
        const key = `${l.side}:${l.anchor}`
        const attached = comments.get(key) || []

        return (
          <div key={i}>
            <div
              className={`flex gap-3 px-4 whitespace-pre border-l-2 font-mono text-[12.5px] leading-[1.75] ${KIND_STYLE[l.kind]}`}
            >
              <span className="w-9 shrink-0 text-right text-faint select-none">
                {l.oldLine || ''}
              </span>
              <span className="w-9 shrink-0 text-right text-faint select-none">
                {l.newLine || ''}
              </span>
              <span
                className={`w-2 shrink-0 select-none ${
                  l.kind === 'add'
                    ? 'text-t4'
                    : l.kind === 'del'
                      ? 'text-t1'
                      : 'text-faint'
                }`}
              >
                {SIGN[l.kind]}
              </span>
              <span className="min-w-0 flex-1">{l.content || ' '}</span>
            </div>

            {attached.map((c) => (
              <Comment key={c.id} c={c} />
            ))}
          </div>
        )
      })}
    </div>
  )
}

function FileBlock({ file, comments }) {
  const [open, setOpen] = useState(true)

  return (
    <section id={`file-${file.path}`} className="mb-5 scroll-mt-20">
      <header className="flex items-center gap-3 px-4 h-11 rounded-t-lg border border-line1 bg-ink2">
        <button
          onClick={() => setOpen((v) => !v)}
          className="font-mono text-[12px] text-faint hover:text-cream w-3"
        >
          {open ? '▾' : '▸'}
        </button>
        <span className="font-mono text-[13px] truncate">{file.path}</span>
        {file.previousPath ? (
          <span className="font-mono text-[11.5px] text-faint truncate">
            was {file.previousPath}
          </span>
        ) : null}
        <span className="ml-auto shrink-0 font-mono text-[11.5px]">
          <span className="text-t4">+{file.additions}</span>{' '}
          <span className="text-t1">−{file.deletions}</span>
        </span>
      </header>

      {open ? (
        <div className="border-x border-b border-line1 rounded-b-lg overflow-x-auto bg-ink">
          {file.binary ? (
            <div className="px-4 py-6 text-[13px] text-faint">
              Binary file not shown.
            </div>
          ) : file.hunks.length === 0 ? (
            <div className="px-4 py-6 text-[13px] text-faint">
              No textual changes.
            </div>
          ) : (
            file.hunks.map((h, i) => (
              <Hunk key={i} hunk={h} comments={comments} />
            ))
          )}
        </div>
      ) : null}
    </section>
  )
}

export default function Review() {
  const { owner, repo, number } = useParams()
  const { loading, data, error } = useAsync(
    () => fetchPullRequest(owner, repo, number),
    [owner, repo, number]
  )

  const commentsByAnchor = useMemo(() => {
    const map = new Map()
    for (const c of data?.comments || []) {
      const key = `${c.path}|${c.side || 'RIGHT'}:${c.line}`
      if (!map.has(key)) map.set(key, [])
      map.get(key).push(c)
    }
    return map
  }, [data])

  if (loading) {
    return <div className="p-6 text-dim text-sm">Loading pull request…</div>
  }
  if (error) {
    return (
      <div className="m-6 p-4 rounded-md border border-c1/40 bg-c1/10 text-[13.5px] text-dim">
        <b className="block mb-1 font-mono text-[11px] uppercase tracking-wider text-t1">
          could not load pull request
        </b>
        {error.message}
      </div>
    )
  }

  const perFile = (path) => {
    const scoped = new Map()
    for (const [key, list] of commentsByAnchor) {
      if (key.startsWith(`${path}|`)) scoped.set(key.slice(path.length + 1), list)
    }
    return scoped
  }

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[210px_minmax(0,1fr)_300px] gap-6 py-6 px-5">
      <aside className="hidden lg:block">
        <div className="sticky top-20">
          <div className="font-mono text-[11px] uppercase tracking-[0.16em] text-faint mb-2">
            {data.files.length} files
          </div>
          <nav className="space-y-0.5 max-h-[75vh] overflow-y-auto pr-1">
            {data.files.map((f) => (
              <a
                key={f.path}
                href={`#file-${f.path}`}
                className="block px-2 py-1 rounded text-[12px] text-dim hover:text-cream hover:bg-ink1 truncate font-mono"
                title={f.path}
              >
                {f.path.split('/').pop()}
              </a>
            ))}
          </nav>
        </div>
      </aside>

      <main className="min-w-0">
        <header className="mb-6">
          <Link
            to={`/app/${owner}/${repo}`}
            className="font-mono text-[12px] text-faint hover:text-cream"
          >
            {owner}/{repo}
          </Link>
          <h1 className="mt-1 text-2xl font-semibold">
            {data.title}{' '}
            <span className="text-faint font-normal">#{data.number}</span>
          </h1>
          <div className="mt-2 flex items-center gap-3 font-mono text-[12px] text-faint">
            <span>{data.author}</span>
            <span className="text-line2">·</span>
            <span className="text-t4">+{data.additions}</span>
            <span className="text-t1">−{data.deletions}</span>
            <span className="text-line2">·</span>
            <span>
              {data.base.ref} ← {data.head.ref}
            </span>
            <a
              href={data.url}
              target="_blank"
              rel="noreferrer"
              className="ml-auto hover:text-cream"
            >
              github ↗
            </a>
          </div>
        </header>

        {data.files.map((f) => (
          <FileBlock key={f.path} file={f} comments={perFile(f.path)} />
        ))}
      </main>

      <aside className="hidden lg:block">
        <div className="sticky top-20 p-4 rounded-lg border border-line1 bg-ink1">
          <div className="font-mono text-[11px] uppercase tracking-[0.16em] text-faint">
            Context
          </div>
          <p className="mt-2 text-[13px] text-dim leading-relaxed">
            Usage lookups arrive once this repository is indexed. Hovering an
            identifier will show everywhere else it appears.
          </p>
        </div>
      </aside>
    </div>
  )
}
