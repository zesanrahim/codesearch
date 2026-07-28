import { Link, Outlet } from 'react-router-dom'

export default function AppShell() {
  return (
    <div className="min-h-screen">
      <nav className="sticky top-0 z-40 h-16 border-b border-line1 bg-ink/85 backdrop-blur-xl">
        <div className="mx-auto max-w-[1400px] px-5 h-full flex items-center gap-8">
          <Link to="/app" className="flex items-center gap-2.5 font-mono text-[15px] font-medium">
            <span className="grid grid-cols-2 gap-[2px]">
              <i className="w-[5px] h-[5px] rounded-[1px] bg-c1" />
              <i className="w-[5px] h-[5px] rounded-[1px] bg-c3" />
              <i className="w-[5px] h-[5px] rounded-[1px] bg-c5" />
              <i className="w-[5px] h-[5px] rounded-[1px] bg-c4" />
            </span>
            codesearch
          </Link>

          <Link
            to="/"
            className="ml-auto text-[13px] text-dim hover:text-cream transition-colors"
          >
            About
          </Link>
        </div>
      </nav>

      <div className="mx-auto max-w-[1400px]">
        <Outlet />
      </div>
    </div>
  )
}
