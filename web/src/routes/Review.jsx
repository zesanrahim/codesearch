import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  createComment,
  deleteComment,
  deleteDraft,
  fetchDrafts,
  fetchIndexStatus,
  fetchPullRequest,
  saveDraft,
  startIndex,
  submitReview,
  useAsync,
} from '../api'
import FileTree from './FileTree'
import useSelectionRange from '../useSelectionRange'

const KIND_STYLE = {
  add: 'bg-c4/[0.14] border-l-c4',
  del: 'bg-c1/[0.12] border-l-c1',
  ctx: 'border-l-transparent',
}

const SIGN = { add: '+', del: '−', ctx: ' ' }

function lineClass(kind, inRange) {
  if (inRange) return 'bg-c5/25 border-l-t5'
  return KIND_STYLE[kind]
}

const anchorKey = (path, side, line) => `${path}|${side}:${line}`

function PostedComment({ c, canDelete, onDelete }) {
  return (
    <div className="my-1.5 mx-10 p-3 rounded-md border border-line1 bg-ink1">
      <div className="flex items-center gap-2 mb-1.5">
        <img src={c.avatarUrl} alt="" className="w-4 h-4 rounded-full" />
        <span className="font-mono text-[11.5px] text-dim">{c.author}</span>
        {c.startLine ? (
          <span className="font-mono text-[10.5px] px-1.5 py-0.5 rounded border border-line2 text-faint">
            lines {c.startLine}–{c.line}
          </span>
        ) : null}
        <div className="ml-auto flex items-center gap-3">
          {canDelete ? (
            <button
              onClick={onDelete}
              className="font-mono text-[11px] text-faint hover:text-t1"
            >
              delete
            </button>
          ) : null}
          <a
            href={c.url}
            target="_blank"
            rel="noreferrer"
            className="font-mono text-[11px] text-faint hover:text-cream"
          >
            github ↗
          </a>
        </div>
      </div>
      <div className="text-[13px] text-cream whitespace-pre-wrap">{c.body}</div>
    </div>
  )
}

function UnsentDraft({ draft, onResume, onDiscard }) {
  return (
    <div className="my-1.5 mx-10 px-3 py-2 rounded-md border border-dashed border-c3/50 bg-c3/[0.06] flex items-center gap-3">
      <span className="font-mono text-[10.5px] uppercase tracking-wider text-t3">
        unsent
      </span>
      <span className="min-w-0 flex-1 truncate text-[12.5px] text-dim">
        {draft.body}
      </span>
      <button
        onClick={onResume}
        className="font-mono text-[11px] text-faint hover:text-cream"
      >
        resume
      </button>
      <button
        onClick={onDiscard}
        className="font-mono text-[11px] text-faint hover:text-t1"
      >
        discard
      </button>
    </div>
  )
}

function Composer({ initial, range, onPost, onCancel, onChange }) {
  const [body, setBody] = useState(initial || '')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState(null)

  const post = async () => {
    if (!body.trim() || busy) return
    setBusy(true)
    setError(null)
    try {
      await onPost(body)
    } catch (e) {
      setError(e.message)
      setBusy(false)
    }
  }

  return (
    <div
      className={`my-1.5 mx-10 p-3 rounded-md bg-ink1 border ${
        range ? 'border-t5/60 border-l-2 border-l-t5' : 'border-line2'
      }`}
    >
      {range ? (
        <div className="mb-2.5 flex items-center gap-2">
          <span className="px-1.5 py-0.5 rounded bg-c5/25 font-mono text-[10.5px] tracking-wide text-t5">
            {range.count} lines
          </span>
          <span className="font-mono text-[11px] text-faint">
            {range.startLine}–{range.line} · {range.side.toLowerCase()}
          </span>
        </div>
      ) : null}
      <textarea
        autoFocus
        value={body}
        onChange={(e) => {
          setBody(e.target.value)
          onChange?.(e.target.value)
        }}
        onKeyDown={(e) => {
          if (e.key === 'Escape') onCancel()
          if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) post()
        }}
        rows={3}
        placeholder={range ? 'Comment on these lines…' : 'Comment on this line…'}
        className="w-full bg-ink border border-line1 rounded p-2 text-[13px] text-cream outline-none focus:border-line2 resize-y"
      />

      {error ? (
        <div className="mt-2 p-2 rounded border border-c1/40 bg-c1/10 text-[12px] text-dim">
          {error}
        </div>
      ) : null}

      <div className="mt-2 flex items-center gap-2">
        <button
          onClick={post}
          disabled={!body.trim() || busy}
          className="h-7 px-3 rounded bg-cream text-ink text-[12px] font-medium disabled:opacity-40"
        >
          {busy ? 'Posting…' : 'Comment'}
        </button>
        <button
          onClick={onCancel}
          disabled={busy}
          className="h-7 px-3 rounded border border-line2 text-[12px] text-dim hover:text-cream"
        >
          Cancel
        </button>
        <span className="ml-auto font-mono text-[10.5px] text-faint">
          ⌘↵ · posts to GitHub
        </span>
      </div>
    </div>
  )
}

function Line({ file, line, posted, draft, composing, range, inRange, viewer, actions }) {
  const key = anchorKey(file.path, line.side, line.anchor)

  return (
    <div>
      <div
        data-path={file.path}
        data-side={line.side}
        data-anchor={line.anchor}
        className={`group flex gap-3 px-4 whitespace-pre border-l-2 font-mono text-[12.5px] leading-[1.75] ${lineClass(line.kind, inRange)}`}
      >
        <span className="w-9 shrink-0 text-right text-faint select-none">
          {line.oldLine || ''}
        </span>
        <span className="w-9 shrink-0 text-right text-faint select-none">
          {line.newLine || ''}
        </span>
        <button
          onClick={() => actions.openComposer(key)}
          title="Comment on this line"
          className="w-4 shrink-0 text-faint opacity-0 group-hover:opacity-100 hover:text-cream"
        >
          +
        </button>
        <span
          className={`w-2 shrink-0 select-none ${
            line.kind === 'add'
              ? 'text-t4'
              : line.kind === 'del'
                ? 'text-t1'
                : 'text-faint'
          }`}
        >
          {SIGN[line.kind]}
        </span>
        <span className="min-w-0 flex-1">{line.content || ' '}</span>
      </div>

      {posted.map((c) => (
        <PostedComment
          key={c.id}
          c={c}
          canDelete={c.author === viewer}
          onDelete={() => actions.removeComment(c.id)}
        />
      ))}

      {draft && !composing ? (
        <UnsentDraft
          draft={draft}
          onResume={() => actions.openComposer(key)}
          onDiscard={() => actions.discardDraft(draft)}
        />
      ) : null}

      {composing ? (
        <Composer
          initial={draft?.body}
          range={range}
          onPost={(body) =>
            actions.post({
              path: file.path,
              line: line.anchor,
              side: line.side,
              body,
            })
          }
          onCancel={actions.closeComposer}
          onChange={(body) =>
            actions.autosave({
              path: file.path,
              line: line.anchor,
              side: line.side,
              body,
            })
          }
        />
      ) : null}
    </div>
  )
}

function FileBlock({ file, postedByKey, draftsByKey, composerKey, pendingRange, viewer, actions }) {
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
            file.hunks.map((h, hi) => (
              <div key={hi}>
                <div className="px-4 py-1 text-[12px] text-faint bg-cream/[0.02] border-y border-line1 font-mono">
                  {h.header}
                </div>
                {h.lines.map((l, li) => {
                  const key = anchorKey(file.path, l.side, l.anchor)
                  const inRange = Boolean(
                    pendingRange &&
                      pendingRange.path === file.path &&
                      pendingRange.side === l.side &&
                      l.anchor >= pendingRange.startLine &&
                      l.anchor <= pendingRange.line
                  )
                  return (
                    <Line
                      key={li}
                      file={file}
                      line={l}
                      posted={postedByKey.get(key) || []}
                      draft={draftsByKey.get(key)}
                      composing={composerKey === key}
                      inRange={inRange}
                      range={
                        composerKey === key && pendingRange?.path === file.path
                          ? pendingRange
                          : null
                      }
                      viewer={viewer}
                      actions={actions}
                    />
                  )
                })}
              </div>
            ))
          )}
        </div>
      ) : null}
    </section>
  )
}

function VerdictModal({ onClose, onSubmit, busy, error }) {
  const [event, setEvent] = useState('APPROVE')
  const [summary, setSummary] = useState('')

  const options = [
    ['APPROVE', 'Approve', 'Approve merging this pull request.'],
    ['REQUEST_CHANGES', 'Request changes', 'Block until feedback is addressed.'],
    ['COMMENT', 'Comment', 'Leave a summary without a verdict.'],
  ]

  const needsSummary = event === 'COMMENT' && !summary.trim()

  return (
    <div className="fixed inset-0 z-50 bg-black/60 flex items-center justify-center p-6">
      <div className="w-full max-w-lg rounded-xl border border-line1 bg-ink1 p-5">
        <h2 className="text-lg font-semibold">Finish review</h2>
        <p className="mt-1 text-[13px] text-dim">
          Inline comments are already posted. This records your verdict.
        </p>

        <div className="mt-4 space-y-1.5">
          {options.map(([value, label, hint]) => (
            <label
              key={value}
              className={`flex gap-3 p-2.5 rounded-md border cursor-pointer ${
                event === value
                  ? 'border-line2 bg-ink2'
                  : 'border-line1 hover:border-line2'
              }`}
            >
              <input
                type="radio"
                name="verdict"
                value={value}
                checked={event === value}
                onChange={() => setEvent(value)}
                className="mt-1 accent-[#146a80]"
              />
              <span>
                <span className="block text-[13.5px]">{label}</span>
                <span className="block text-[12px] text-faint">{hint}</span>
              </span>
            </label>
          ))}
        </div>

        <textarea
          value={summary}
          onChange={(e) => setSummary(e.target.value)}
          rows={3}
          placeholder={
            event === 'COMMENT' ? 'Summary (required)' : 'Summary (optional)'
          }
          className="mt-4 w-full bg-ink border border-line1 rounded p-2 text-[13px] text-cream outline-none focus:border-line2 resize-y"
        />

        {error ? (
          <div className="mt-3 p-2.5 rounded border border-c1/40 bg-c1/10 text-[12.5px] text-dim">
            {error}
          </div>
        ) : null}

        <div className="mt-4 flex gap-2">
          <button
            onClick={() => onSubmit({ event, summary })}
            disabled={busy || needsSummary}
            className="h-9 px-4 rounded-md bg-cream text-ink text-[13px] font-medium disabled:opacity-50"
          >
            {busy ? 'Submitting…' : 'Submit'}
          </button>
          <button
            onClick={onClose}
            disabled={busy}
            className="h-9 px-4 rounded-md border border-line2 text-[13px] text-dim hover:text-cream"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  )
}

const ACTIVE = new Set(['queued', 'cloning', 'indexing'])

const LABEL = {
  absent: 'not indexed',
  queued: 'queued',
  cloning: 'cloning repository…',
  indexing: 'indexing',
  ready: 'index ready',
  failed: 'indexing failed',
}

function ContextPanel({ indexed, owner, repo }) {
  const [job, setJob] = useState(indexed ? { status: 'ready' } : null)

  useEffect(() => {
    let stopped = false
    let timer

    const tick = async () => {
      try {
        const next = await fetchIndexStatus(owner, repo)
        if (stopped) return
        setJob(next)
        if (ACTIVE.has(next.status)) timer = setTimeout(tick, 700)
      } catch {
        if (!stopped) setJob({ status: 'absent' })
      }
    }

    tick()
    return () => {
      stopped = true
      clearTimeout(timer)
    }
  }, [owner, repo])

  const status = job?.status || 'absent'
  const pct =
    status === 'indexing' && job.total > 0
      ? Math.round((job.processed / job.total) * 100)
      : null

  const dot =
    status === 'ready'
      ? 'bg-c4'
      : status === 'failed'
        ? 'bg-c1'
        : ACTIVE.has(status)
          ? 'bg-c3 animate-pulse'
          : 'bg-line2'

  const retry = async () => {
    setJob({ status: 'queued' })
    try {
      setJob(await startIndex(owner, repo))
    } catch {
      setJob({ status: 'failed', error: 'could not start indexing' })
    }
  }

  return (
    <div className="sticky top-20 p-4 rounded-lg border border-line1 bg-ink1">
      <div className="font-mono text-[11px] uppercase tracking-[0.16em] text-faint">
        Context
      </div>

      <div className="mt-3 flex items-center gap-2 text-[12.5px]">
        <span className={`w-1.5 h-1.5 rounded-full shrink-0 ${dot}`} />
        <span
          className={
            status === 'ready'
              ? 'text-t4'
              : status === 'failed'
                ? 'text-t1'
                : ACTIVE.has(status)
                  ? 'text-t3'
                  : 'text-faint'
          }
        >
          {LABEL[status] || status}
        </span>
        {pct !== null ? (
          <span className="ml-auto font-mono text-[11px] text-faint">
            {pct}%
          </span>
        ) : null}
      </div>

      {status === 'indexing' && job.total > 0 ? (
        <>
          <div className="mt-2.5 h-1 rounded-full bg-ink overflow-hidden">
            <div
              className="h-full bg-c3 transition-[width] duration-300"
              style={{ width: `${pct}%` }}
            />
          </div>
          <p className="mt-1.5 font-mono text-[11px] text-faint">
            {job.processed.toLocaleString()} / {job.total.toLocaleString()} files
          </p>
        </>
      ) : null}

      {status === 'ready' && job?.files ? (
        <p className="mt-1.5 font-mono text-[11px] text-faint">
          {job.files.toLocaleString()} files indexed
        </p>
      ) : null}

      {status === 'failed' ? (
        <p className="mt-2 text-[12px] text-dim break-words">{job.error}</p>
      ) : null}

      {(status === 'absent' || status === 'failed') ? (
        <button
          onClick={retry}
          className="mt-3 h-7 px-3 rounded border border-line2 text-[12px] text-cream hover:bg-cream/5"
        >
          {status === 'failed' ? 'Retry' : 'Index now'}
        </button>
      ) : null}

      <p className="mt-3 pt-3 border-t border-line1 text-[12.5px] text-dim leading-relaxed">
        {status === 'ready'
          ? 'Identifier usage lookups land next; the index they need is ready.'
          : 'Identifier usage lookups need this index.'}
      </p>
    </div>
  )
}

export default function Review() {
  const { owner, repo, number } = useParams()
  const [activeFile, setActiveFile] = useState(null)
  const [drafts, setDrafts] = useState([])
  const [composerKey, setComposerKey] = useState(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [submitError, setSubmitError] = useState(null)
  const [pendingRange, setPendingRange] = useState(null)
  const autosaveTimer = useRef(null)

  const { loading, data, error, reload } = useAsync(
    () => fetchPullRequest(owner, repo, number),
    [owner, repo, number]
  )

  useEffect(() => {
    fetchDrafts(owner, repo, number)
      .then((s) => setDrafts(s.drafts || []))
      .catch(() => setDrafts([]))
  }, [owner, repo, number])

  const postedByKey = useMemo(() => {
    const map = new Map()
    for (const c of data?.comments || []) {
      const key = anchorKey(c.path, c.side || 'RIGHT', c.line)
      if (!map.has(key)) map.set(key, [])
      map.get(key).push(c)
    }
    return map
  }, [data])

  const draftsByKey = useMemo(() => {
    const map = new Map()
    for (const d of drafts) map.set(anchorKey(d.path, d.side, d.line), d)
    return map
  }, [drafts])

  const headSHA = data?.head?.sha

  const actions = useMemo(
    () => ({
      openComposer: (key) => setComposerKey(key),
      closeComposer: () => {
        setComposerKey(null)
        setPendingRange(null)
      },

      post: async (comment) => {
        const range =
          pendingRange &&
          pendingRange.path === comment.path &&
          pendingRange.line === comment.line &&
          pendingRange.side === comment.side
            ? pendingRange
            : null

        await createComment(owner, repo, number, {
          ...comment,
          startLine: range?.startLine,
          startSide: range?.startSide,
          commitId: headSHA,
        })
        setPendingRange(null)
        setDrafts((prev) =>
          prev.filter(
            (d) =>
              !(
                d.path === comment.path &&
                d.line === comment.line &&
                d.side === comment.side
              )
          )
        )
        setComposerKey(null)
        reload()
      },

      autosave: (draft) => {
        clearTimeout(autosaveTimer.current)
        if (!draft.body.trim()) return
        autosaveTimer.current = setTimeout(() => {
          saveDraft(owner, repo, number, draft)
            .then((s) => setDrafts(s.drafts || []))
            .catch(() => {})
        }, 700)
      },

      discardDraft: async (draft) => {
        const s = await deleteDraft(owner, repo, number, draft)
        setDrafts(s.drafts || [])
      },

      removeComment: async (id) => {
        await deleteComment(owner, repo, number, id)
        reload()
      },
    }),
    [owner, repo, number, headSHA, reload, pendingRange]
  )

  const onRange = useCallback((range) => {
    setPendingRange(range)
    setComposerKey(anchorKey(range.path, range.side, range.line))
  }, [])

  useSelectionRange(onRange)

  useEffect(() => {
    const onKey = (e) => {
      if (e.key !== 'Escape') return
      setPendingRange(null)
      setComposerKey(null)
    }
    document.addEventListener('keydown', onKey)
    return () => document.removeEventListener('keydown', onKey)
  }, [])

  const onSubmit = useCallback(
    async ({ event, summary }) => {
      setBusy(true)
      setSubmitError(null)
      try {
        await submitReview(owner, repo, number, { event, summary })
        setModalOpen(false)
        reload()
      } catch (e) {
        setSubmitError(e.message)
      } finally {
        setBusy(false)
      }
    },
    [owner, repo, number, reload]
  )

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

  return (
    <div className="grid grid-cols-1 lg:grid-cols-[230px_minmax(0,1fr)_290px] gap-6 py-6 px-5">
      <aside className="hidden lg:block">
        <div className="sticky top-20">
          <div className="font-mono text-[11px] uppercase tracking-[0.16em] text-faint mb-2 px-1.5">
            {data.files.length} files
          </div>
          <nav className="max-h-[75vh] overflow-y-auto pr-1">
            <FileTree
              files={data.files}
              active={activeFile}
              onSelect={setActiveFile}
            />
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

          <div className="mt-3 flex items-center gap-3 font-mono text-[12px] text-faint">
            <span>{data.author}</span>
            <span className="text-line2">·</span>
            <span className="text-t4">+{data.additions}</span>
            <span className="text-t1">−{data.deletions}</span>
            <span className="text-line2">·</span>
            <span>
              {data.base.ref} ← {data.head.ref}
            </span>

            <div className="ml-auto flex items-center gap-3">
              <a
                href={data.url}
                target="_blank"
                rel="noreferrer"
                className="hover:text-cream"
              >
                github ↗
              </a>
              <button
                onClick={() => setModalOpen(true)}
                className="h-8 px-3 rounded-md border border-line2 text-[12.5px] font-sans text-cream hover:bg-cream/5"
              >
                Finish review
              </button>
            </div>
          </div>
        </header>

        {data.files.map((f) => (
          <FileBlock
            key={f.path}
            file={f}
            postedByKey={postedByKey}
            draftsByKey={draftsByKey}
            composerKey={composerKey}
            pendingRange={pendingRange}
            viewer={data.viewer}
            actions={actions}
          />
        ))}
      </main>

      <aside className="hidden lg:block">
        <ContextPanel indexed={data.indexed} owner={owner} repo={repo} />
      </aside>

      {modalOpen ? (
        <VerdictModal
          busy={busy}
          error={submitError}
          onClose={() => setModalOpen(false)}
          onSubmit={onSubmit}
        />
      ) : null}
    </div>
  )
}
