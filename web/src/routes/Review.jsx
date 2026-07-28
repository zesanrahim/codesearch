import { useCallback, useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import {
  deleteDraft,
  fetchDrafts,
  fetchPullRequest,
  saveDraft,
  submitReview,
  useAsync,
} from '../api'
import FileTree from './FileTree'

const KIND_STYLE = {
  add: 'bg-c4/[0.14] border-l-c4',
  del: 'bg-c1/[0.12] border-l-c1',
  ctx: 'border-l-transparent',
}

const SIGN = { add: '+', del: '−', ctx: ' ' }

const anchorKey = (path, side, line) => `${path}|${side}:${line}`

function PostedComment({ c }) {
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
      <div className="text-[13px] text-cream whitespace-pre-wrap">{c.body}</div>
    </div>
  )
}

function DraftComment({ draft, onEdit, onDelete }) {
  return (
    <div className="my-1.5 mx-10 p-3 rounded-md border border-c3/40 bg-c3/[0.08]">
      <div className="flex items-center gap-2 mb-1.5">
        <span className="font-mono text-[10.5px] uppercase tracking-wider text-t3">
          pending
        </span>
        <button
          onClick={onEdit}
          className="ml-auto font-mono text-[11px] text-faint hover:text-cream"
        >
          edit
        </button>
        <button
          onClick={onDelete}
          className="font-mono text-[11px] text-faint hover:text-t1"
        >
          discard
        </button>
      </div>
      <div className="text-[13px] text-cream whitespace-pre-wrap">
        {draft.body}
      </div>
    </div>
  )
}

function Composer({ initial, onSave, onCancel }) {
  const [body, setBody] = useState(initial || '')

  const submit = () => {
    if (body.trim()) onSave(body)
  }

  return (
    <div className="my-1.5 mx-10 p-3 rounded-md border border-line2 bg-ink1">
      <textarea
        autoFocus
        value={body}
        onChange={(e) => setBody(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') onCancel()
          if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) submit()
        }}
        rows={3}
        placeholder="Leave a comment…"
        className="w-full bg-ink border border-line1 rounded p-2 text-[13px] text-cream outline-none focus:border-line2 resize-y"
      />
      <div className="mt-2 flex items-center gap-2">
        <button
          onClick={submit}
          disabled={!body.trim()}
          className="h-7 px-3 rounded bg-cream text-ink text-[12px] font-medium disabled:opacity-40"
        >
          Add comment
        </button>
        <button
          onClick={onCancel}
          className="h-7 px-3 rounded border border-line2 text-[12px] text-dim hover:text-cream"
        >
          Cancel
        </button>
        <span className="ml-auto font-mono text-[10.5px] text-faint">⌘↵</span>
      </div>
    </div>
  )
}

function Line({ file, line, posted, draft, composing, actions }) {
  const key = anchorKey(file.path, line.side, line.anchor)

  return (
    <div>
      <div
        className={`group flex gap-3 px-4 whitespace-pre border-l-2 font-mono text-[12.5px] leading-[1.75] ${KIND_STYLE[line.kind]}`}
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
        <PostedComment key={c.id} c={c} />
      ))}

      {draft && !composing ? (
        <DraftComment
          draft={draft}
          onEdit={() => actions.openComposer(key)}
          onDelete={() => actions.remove(draft)}
        />
      ) : null}

      {composing ? (
        <Composer
          initial={draft?.body}
          onSave={(body) =>
            actions.save({
              path: file.path,
              line: line.anchor,
              side: line.side,
              body,
            })
          }
          onCancel={actions.closeComposer}
        />
      ) : null}
    </div>
  )
}

function FileBlock({ file, postedByKey, draftsByKey, composerKey, actions }) {
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
                  return (
                    <Line
                      key={li}
                      file={file}
                      line={l}
                      posted={postedByKey.get(key) || []}
                      draft={draftsByKey.get(key)}
                      composing={composerKey === key}
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

function SubmitModal({ drafts, onClose, onSubmit, busy, error }) {
  const [event, setEvent] = useState('COMMENT')
  const [summary, setSummary] = useState('')

  const options = [
    ['COMMENT', 'Comment', 'Leave feedback without a verdict.'],
    ['APPROVE', 'Approve', 'Submit feedback and approve merging.'],
    ['REQUEST_CHANGES', 'Request changes', 'Feedback that must be addressed.'],
  ]

  return (
    <div className="fixed inset-0 z-50 bg-black/60 flex items-center justify-center p-6">
      <div className="w-full max-w-lg rounded-xl border border-line1 bg-ink1 p-5">
        <h2 className="text-lg font-semibold">Submit review</h2>
        <p className="mt-1 text-[13px] text-dim">
          {drafts.length} comment{drafts.length === 1 ? '' : 's'} will be posted
          to GitHub.
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
                name="event"
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
          placeholder="Summary (optional)"
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
            disabled={busy}
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

export default function Review() {
  const { owner, repo, number } = useParams()
  const [activeFile, setActiveFile] = useState(null)
  const [drafts, setDrafts] = useState([])
  const [composerKey, setComposerKey] = useState(null)
  const [modalOpen, setModalOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [submitError, setSubmitError] = useState(null)

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

  const actions = useMemo(
    () => ({
      openComposer: (key) => setComposerKey(key),
      closeComposer: () => setComposerKey(null),
      save: async (draft) => {
        const state = await saveDraft(owner, repo, number, draft)
        setDrafts(state.drafts || [])
        setComposerKey(null)
      },
      remove: async (draft) => {
        const state = await deleteDraft(owner, repo, number, draft)
        setDrafts(state.drafts || [])
      },
    }),
    [owner, repo, number]
  )

  const onSubmit = useCallback(
    async ({ event, summary }) => {
      setBusy(true)
      setSubmitError(null)
      try {
        await submitReview(owner, repo, number, { event, summary })
        setDrafts([])
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
    <div className="pb-20">
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
            <FileBlock
              key={f.path}
              file={f}
              postedByKey={postedByKey}
              draftsByKey={draftsByKey}
              composerKey={composerKey}
              actions={actions}
            />
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

      {drafts.length > 0 ? (
        <div className="fixed bottom-0 inset-x-0 z-40 border-t border-line1 bg-ink/95 backdrop-blur-xl">
          <div className="mx-auto max-w-[1400px] px-5 h-14 flex items-center gap-4">
            <span className="text-[13px] text-dim">
              <b className="text-cream font-semibold">{drafts.length}</b> comment
              {drafts.length === 1 ? '' : 's'} pending
            </span>
            <button
              onClick={() => setModalOpen(true)}
              className="ml-auto h-9 px-4 rounded-md bg-cream text-ink text-[13px] font-medium hover:bg-white"
            >
              Submit review
            </button>
          </div>
        </div>
      ) : null}

      {modalOpen ? (
        <SubmitModal
          drafts={drafts}
          busy={busy}
          error={submitError}
          onClose={() => setModalOpen(false)}
          onSubmit={onSubmit}
        />
      ) : null}
    </div>
  )
}
