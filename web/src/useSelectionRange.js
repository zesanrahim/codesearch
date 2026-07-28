import { useEffect, useState } from 'react'

function lineElement(node) {
  if (!node) return null
  const el = node.nodeType === Node.TEXT_NODE ? node.parentElement : node
  return el?.closest?.('[data-anchor]') || null
}

function orderedPair(a, b) {
  const following =
    a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING
  return following ? [a, b] : [b, a]
}

function resolve(selection) {
  if (!selection || selection.isCollapsed) return null
  if (!selection.toString().trim()) return null

  const anchor = lineElement(selection.anchorNode)
  const focus = lineElement(selection.focusNode)
  if (!anchor || !focus) return null

  const path = anchor.dataset.path
  if (path !== focus.dataset.path) return null

  const [first, last] = orderedPair(anchor, focus)
  const side = last.dataset.side

  const section = last.closest('section')
  if (!section) return null

  const all = [...section.querySelectorAll('[data-anchor]')]
  const from = all.indexOf(first)
  const to = all.indexOf(last)
  if (from < 0 || to < 0) return null

  const spanned = all
    .slice(from, to + 1)
    .filter((el) => el.dataset.side === side)
    .map((el) => Number(el.dataset.anchor))

  if (spanned.length === 0) return null

  const startLine = Math.min(...spanned)
  const endLine = Math.max(...spanned)
  if (startLine === endLine) return null

  const rect = selection.getRangeAt(0).getBoundingClientRect()

  return {
    path,
    side,
    startSide: side,
    startLine,
    line: endLine,
    count: endLine - startLine + 1,
    rect: { top: rect.top, left: rect.left, bottom: rect.bottom },
  }
}

export default function useSelectionRange() {
  const [range, setRange] = useState(null)

  useEffect(() => {
    const update = () => setRange(resolve(document.getSelection()))

    const onKey = (e) => {
      if (e.key === 'Escape') setRange(null)
    }

    document.addEventListener('mouseup', update)
    document.addEventListener('keyup', onKey)
    return () => {
      document.removeEventListener('mouseup', update)
      document.removeEventListener('keyup', onKey)
    }
  }, [])

  return [range, () => setRange(null)]
}
