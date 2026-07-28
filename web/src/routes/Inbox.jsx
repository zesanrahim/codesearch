import { Link, useParams } from 'react-router-dom'
import { fetchInbox, since, useAsync } from '../api'

function Row({ item }) {
  return (
    <Link
      to={`/app/${item.owner}/${item.repo}/pull/${item.number}`}
      className="flex items-center gap-4 px-5 py-3.5 border-b border-line1 hover:bg-ink1 transition-colors"
    >
      <img
        src={item.avatarUrl}
        alt=""
        className="w-6 h-6 rounded-full shrink-0 opacity-90"
      />

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="truncate text-[15px]">{item.title}</span>
          {item.draft ? (
            <span className="shrink-0 px-1.5 py-0.5 rounded border border-line2 font-mono text-[10px] uppercase tracking-wider text-faint">
              draft
            </span>
          ) : null}
        </div>
        <div className="mt-0.5 font-mono text-[12px] text-faint">
          {item.owner}/{item.repo}
          <span className="text-line2"> · </span>#{item.number}
          <span className="text-line2"> · </span>
          {item.author}
        </div>
      </div>

      {item.indexed ? (
        <span
          className="shrink-0 w-1.5 h-1.5 rounded-full bg-c4"
          title="indexed — usage data available"
        />
      ) : null}

      <span className="shrink-0 font-mono text-[12px] text-faint w-10 text-right">
        {since(item.updatedAt)}
      </span>
    </Link>
  )
}

function Section({ title, items }) {
  if (!items.length) return null

  return (
    <section className="mb-9">
      <h2 className="px-5 pb-2 font-mono text-[11px] uppercase tracking-[0.16em] text-faint">
        {title}
        <span className="ml-2 text-line2">{items.length}</span>
      </h2>
      <div className="border-t border-line1">
        {items.map((i) => (
          <Row key={`${i.owner}/${i.repo}#${i.number}`} item={i} />
        ))}
      </div>
    </section>
  )
}

export default function Inbox() {
  const { owner, repo } = useParams()
  const scoped = Boolean(owner && repo)

  const { loading, data, error } = useAsync(
    () => fetchInbox(scoped ? { owner, repo } : {}),
    [owner, repo]
  )

  if (loading) {
    return <div className="p-6 text-dim text-sm">Loading pull requests…</div>
  }
  if (error) {
    return (
      <div className="m-6 p-4 rounded-md border border-c1/40 bg-c1/10 text-[13.5px] text-dim">
        <b className="block mb-1 font-mono text-[11px] uppercase tracking-wider text-t1">
          could not load inbox
        </b>
        {error.message}
      </div>
    )
  }

  const items = data.items || []
  const mine = items.filter((i) => i.author === data.viewer)
  const others = items.filter((i) => i.author !== data.viewer)

  return (
    <div className="py-6">
      <header className="px-5 pb-6 flex items-baseline gap-3">
        <h1 className="text-2xl font-semibold">
          {scoped ? `${owner}/${repo}` : 'Open pull requests'}
        </h1>
        <span className="font-mono text-[12px] text-faint">
          {items.length} open
        </span>
        {scoped ? (
          <Link to="/app" className="ml-auto text-[13px] text-dim hover:text-cream">
            ← all repositories
          </Link>
        ) : null}
      </header>

      {items.length === 0 ? (
        <p className="px-5 text-dim text-sm">
          Nothing open here right now.
        </p>
      ) : (
        <>
          <Section title="Yours" items={mine} />
          <Section title="Everything else" items={others} />
        </>
      )}
    </div>
  )
}
