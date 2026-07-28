import { useEffect, useRef, useState, lazy, Suspense } from 'react'
import { Link, useLocation, useNavigate } from 'react-router-dom'
import DiffDemo from './DiffDemo'

const Scene3D = lazy(() => import('./Scene3D'))

const steps = [
  {
    n: '01',
    title: 'Point it at a pull request',
    body: 'It fetches the diff and indexes the base commit — the codebase as it exists without your change.',
  },
  {
    n: '02',
    title: 'Every identifier becomes a link',
    body: 'Exact, word-boundary lookups. Save does not match SaveConfig, and err is suppressed as noise.',
  },
  {
    n: '03',
    title: 'Read the diff in context',
    body: 'Hover to pull up the other call sites, then judge whether the new code matches them.',
  },
]

const ROUTES = [
  { path: '/', id: 'top' },
  { path: '/demo', id: 'demo' },
  { path: '/how', id: 'how' },
]

function scrollToId(id, behavior = 'smooth') {
  if (id === 'top') {
    window.scrollTo({ top: 0, behavior })
    return
  }
  document.getElementById(id)?.scrollIntoView({ behavior, block: 'start' })
}

function SectionLink({ to, className, children }) {
  const navigate = useNavigate()
  const route = ROUTES.find((r) => r.path === to)

  const go = (e) => {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return
    e.preventDefault()
    navigate(to)
    scrollToId(route?.id ?? 'top')
  }

  return (
    <a href={to} onClick={go} className={className}>
      {children}
    </a>
  )
}

function useSectionRoutes() {
  const location = useLocation()

  useEffect(() => {
    const match = ROUTES.find((r) => r.path === location.pathname)
    if (match && match.id !== 'top') {
      requestAnimationFrame(() => scrollToId(match.id, 'auto'))
    }
  }, [location.pathname])
}

function Brand() {
  return (
    <div className="flex items-center gap-2.5 font-mono text-[15px] font-medium">
      <div className="grid grid-cols-2 gap-[2px]">
        <i className="w-[5px] h-[5px] rounded-[1px] bg-c1" />
        <i className="w-[5px] h-[5px] rounded-[1px] bg-c3" />
        <i className="w-[5px] h-[5px] rounded-[1px] bg-c5" />
        <i className="w-[5px] h-[5px] rounded-[1px] bg-c4" />
      </div>
      codesearch
    </div>
  )
}

function Nav() {
  const nav = useRef(null)

  useEffect(() => {
    const el = nav.current
    if (!el) return

    const move = (e) => {
      const r = el.getBoundingClientRect()
      el.style.setProperty('--mx', `${e.clientX - r.left}px`)
    }

    window.addEventListener('mousemove', move)
    return () => window.removeEventListener('mousemove', move)
  }, [])

  return (
    <nav
      ref={nav}
      className="sticky top-0 z-40 h-16 border-b border-line1 bg-ink/80 backdrop-blur-xl relative"
    >
      <div className="absolute inset-0 navglow pointer-events-none" />
      <div className="relative mx-auto max-w-[1120px] px-6 h-full flex items-center gap-8">
        <Brand />
        <div className="hidden md:flex gap-7 text-sm text-dim">
          <SectionLink className="hover:text-cream transition-colors" to="/how">
            How it works
          </SectionLink>
          <SectionLink className="hover:text-cream transition-colors" to="/demo">
            Demo
          </SectionLink>
          <a
            className="hover:text-cream transition-colors"
            href="https://github.com/zesanrahim/codesearch"
          >
            GitHub
          </a>
        </div>
      </div>
    </nav>
  )
}

function useLevel() {
  const ref = useRef(null)
  const [level, setLevel] = useState(false)

  useEffect(() => {
    const el = ref.current
    if (!el) return

    const io = new IntersectionObserver(
      ([entry]) => entry.isIntersecting && setLevel(true),
      { threshold: 0.15 }
    )
    io.observe(el)
    return () => io.disconnect()
  }, [])

  return [ref, level]
}

export default function App() {
  const [demoRef, level] = useLevel()
  useSectionRoutes()

  return (
    <div className="relative overflow-x-hidden">
      <div className="pointer-events-none fixed inset-0 z-0 dotgrid" />

      <div className="relative z-10">
        <Nav />

        <header className="relative pt-24 pb-16">
          <div className="pointer-events-none absolute inset-x-0 -top-24 h-[680px] opacity-70">
            <Suspense fallback={null}>
              <Scene3D />
            </Suspense>
          </div>
          <div
            className="pointer-events-none absolute inset-x-0 -top-32 h-[640px]"
            style={{
              background:
                'radial-gradient(42% 46% at 22% 30%, rgba(20,106,128,0.26), transparent 70%), radial-gradient(38% 42% at 78% 14%, rgba(192,137,17,0.14), transparent 72%)',
            }}
          />

          <div className="relative mx-auto max-w-[1120px] px-6">
            <div className="max-w-[760px]">
              <span className="inline-flex items-center gap-2.5 font-mono text-[11px] tracking-[0.16em] uppercase text-faint">
                <i className="w-1.5 h-1.5 rounded-[1px] bg-t3" />
                Pull request pattern review
              </span>

              <h1 className="mt-5 text-[clamp(2.5rem,6vw,4.15rem)] font-semibold leading-[1.05]">
                Review the patterns,
                <br />
                not just the <span className="text-t3">diff</span>.
              </h1>

              <p className="mt-6 max-w-[600px] text-[18px] leading-relaxed text-dim">
                A diff tells you what changed. It does not tell you whether the
                new code looks like the code you already have. Hover any
                identifier to see everywhere else it appears.
              </p>

              <div className="mt-8 flex flex-wrap gap-3">
                <Link
                  to="/app"
                  className="h-10 px-5 rounded-md bg-cream text-ink text-sm font-medium flex items-center hover:bg-white hover:-translate-y-px transition-all"
                >
                  Open the inbox
                </Link>
                <SectionLink
                  to="/how"
                  className="h-10 px-5 rounded-md border border-line2 text-sm font-medium flex items-center hover:border-faint hover:bg-cream/5 transition-all"
                >
                  How it works
                </SectionLink>
              </div>

              <p className="mt-5 font-mono text-[12.5px] text-faint">
                No inference. No invented citations. A trigram index and a token
                map.
              </p>
            </div>

            <div
              ref={demoRef}
              className={`mt-14 tilt ${level ? 'level' : ''}`}
            >
              <DiffDemo />
            </div>
          </div>
        </header>

        <section id="how" className="py-24 border-t border-line1">
          <div className="mx-auto max-w-[1120px] px-6">
            <div className="max-w-[640px]">
              <span className="inline-flex items-center gap-2.5 font-mono text-[11px] tracking-[0.16em] uppercase text-faint">
                <i className="w-1.5 h-1.5 rounded-[1px] bg-t5" />
                How it works
              </span>
              <h2 className="mt-4 text-[clamp(1.75rem,3.4vw,2.4rem)] font-semibold">
                Three steps, no model in the loop.
              </h2>
              <p className="mt-4 text-[16.5px] leading-relaxed text-dim">
                The whole thing is a lookup. That is what makes it instant
                enough to fire on every mouse move, and why it can never cite a
                file that is not there.
              </p>
            </div>

            <div className="mt-11 grid grid-cols-1 md:grid-cols-3 gap-4">
              {steps.map((s) => (
                <div
                  key={s.n}
                  className="p-6 rounded-lg border border-line1 bg-ink1 hover:border-line2 hover:-translate-y-0.5 transition-all"
                >
                  <div className="font-mono text-[11px] tracking-[0.14em] text-faint">
                    {s.n}
                  </div>
                  <h3 className="mt-3.5 text-[16.5px] font-semibold">
                    {s.title}
                  </h3>
                  <p className="mt-2 text-[14.5px] leading-relaxed text-dim">
                    {s.body}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </section>

        <section className="py-28 border-t border-line1 text-center">
          <div className="mx-auto max-w-[1120px] px-6">
            <h2 className="text-[clamp(1.9rem,4vw,2.8rem)] font-semibold">
              Stop reviewing diffs in isolation.
            </h2>
            <p className="mt-5 mx-auto max-w-[520px] text-dim leading-relaxed">
              Point it at one pull request and see whether the extra context
              changes what you would have said.
            </p>
            <div className="mt-8 flex justify-center gap-3">
              <Link
                to="/app"
                className="h-10 px-5 rounded-md bg-cream text-ink text-sm font-medium flex items-center hover:bg-white hover:-translate-y-px transition-all"
              >
                Open the inbox
              </Link>
              <SectionLink
                to="/demo"
                className="h-10 px-5 rounded-md border border-line2 text-sm font-medium flex items-center hover:border-faint hover:bg-cream/5 transition-all"
              >
                See the demo
              </SectionLink>
            </div>
          </div>
        </section>

        <footer className="border-t border-line1 py-9">
          <div className="mx-auto max-w-[1120px] px-6 flex flex-wrap items-center gap-4 text-[13px] text-faint">
            <Brand />
            <span>Trigram code search, deterministic review.</span>
            <div className="ml-auto flex gap-5">
              <a
                className="hover:text-cream transition-colors"
                href="https://github.com/zesanrahim/codesearch"
              >
                GitHub
              </a>
              <SectionLink
                className="hover:text-cream transition-colors"
                to="/how"
              >
                How it works
              </SectionLink>
            </div>
          </div>
        </footer>
      </div>
    </div>
  )
}
