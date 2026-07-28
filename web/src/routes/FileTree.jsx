import { useState } from 'react'

function emptyNode() {
  return { dirs: new Map(), files: [] }
}

function buildTree(files) {
  const root = emptyNode()

  for (const file of files) {
    const parts = file.path.split('/')
    const name = parts.pop()

    let node = root
    for (const part of parts) {
      if (!node.dirs.has(part)) node.dirs.set(part, emptyNode())
      node = node.dirs.get(part)
    }
    node.files.push({ ...file, name })
  }

  return collapse(root)
}

function collapse(node) {
  const dirs = new Map()

  for (const [name, child] of node.dirs) {
    let label = name
    let current = collapse(child)

    while (current.files.length === 0 && current.dirs.size === 1) {
      const [onlyName, onlyChild] = [...current.dirs.entries()][0]
      label = `${label}/${onlyName}`
      current = onlyChild
    }

    dirs.set(label, current)
  }

  return { dirs, files: node.files }
}

function Leaf({ file, active, onSelect }) {
  const jump = (e) => {
    e.preventDefault()
    onSelect?.(file.path)
    document
      .getElementById(`file-${file.path}`)
      ?.scrollIntoView({ behavior: 'auto', block: 'start' })
  }

  return (
    <a
      href={`#file-${file.path}`}
      onClick={jump}
      title={file.path}
      className={`group flex items-center gap-2 pl-2 pr-1.5 py-1 rounded font-mono text-[12px] hover:bg-ink2 ${
        active ? 'bg-ink2 text-cream' : 'text-dim'
      }`}
    >
      <span className="truncate flex-1">{file.name}</span>
      <span className="shrink-0 text-[10.5px] opacity-70 group-hover:opacity-100">
        <span className="text-t4">+{file.additions}</span>
        <span className="text-t1 ml-1">−{file.deletions}</span>
      </span>
    </a>
  )
}

function Dir({ name, node, depth, active, onSelect }) {
  const [open, setOpen] = useState(true)

  return (
    <div>
      <button
        onClick={() => setOpen((v) => !v)}
        className="w-full flex items-center gap-1.5 px-1.5 py-1 rounded font-mono text-[11.5px] text-faint hover:text-dim hover:bg-ink1"
        style={{ paddingLeft: depth * 10 + 6 }}
      >
        <span className="w-2 shrink-0">{open ? '▾' : '▸'}</span>
        <span className="truncate">{name}</span>
      </button>

      {open ? (
        <Branch node={node} depth={depth + 1} active={active} onSelect={onSelect} />
      ) : null}
    </div>
  )
}

function Branch({ node, depth, active, onSelect }) {
  return (
    <div>
      {[...node.dirs.entries()].map(([name, child]) => (
        <Dir
          key={name}
          name={name}
          node={child}
          depth={depth}
          active={active}
          onSelect={onSelect}
        />
      ))}

      {node.files.map((file) => (
        <div key={file.path} style={{ paddingLeft: depth * 10 }}>
          <Leaf file={file} active={active === file.path} onSelect={onSelect} />
        </div>
      ))}
    </div>
  )
}

export default function FileTree({ files, active, onSelect }) {
  const tree = buildTree(files)
  return <Branch node={tree} depth={0} active={active} onSelect={onSelect} />
}
