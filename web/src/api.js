import { useCallback, useEffect, useState } from 'react'

async function json(url, options) {
  const res = await fetch(url, options)
  const body = await res.json().catch(() => ({}))
  if (!res.ok) throw new Error(body.error || `HTTP ${res.status}`)
  return body
}

export function fetchInbox({ owner, repo, org } = {}) {
  const q = new URLSearchParams()
  if (owner && repo) {
    q.set('owner', owner)
    q.set('repo', repo)
  } else if (org) {
    q.set('org', org)
  }
  return json(`/api/inbox?${q.toString()}`)
}

export function fetchPullRequest(owner, repo, number) {
  return json(`/api/pr/${owner}/${repo}/${number}`)
}

const asJSON = (method, body) => ({
  method,
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify(body),
})

export function fetchDrafts(owner, repo, number) {
  return json(`/api/pr/${owner}/${repo}/${number}/drafts`)
}

export function saveDraft(owner, repo, number, draft) {
  return json(`/api/pr/${owner}/${repo}/${number}/drafts`, asJSON('PUT', draft))
}

export function deleteDraft(owner, repo, number, { path, line, side }) {
  const q = new URLSearchParams({ path, line: String(line), side })
  return json(`/api/pr/${owner}/${repo}/${number}/drafts?${q}`, {
    method: 'DELETE',
  })
}

export function createComment(owner, repo, number, comment) {
  return json(
    `/api/pr/${owner}/${repo}/${number}/comments`,
    asJSON('POST', comment)
  )
}

export function deleteComment(owner, repo, number, id) {
  return json(`/api/pr/${owner}/${repo}/${number}/comments/${id}`, {
    method: 'DELETE',
  })
}

export function submitReview(owner, repo, number, { event, summary }) {
  return json(
    `/api/pr/${owner}/${repo}/${number}/review`,
    asJSON('POST', { event, summary })
  )
}

export function fetchIndexStatus(owner, repo) {
  return json(`/api/index/${owner}/${repo}`)
}

export function startIndex(owner, repo) {
  return json(`/api/index/${owner}/${repo}`, { method: 'POST' })
}

export function useAsync(fn, deps) {
  const [state, setState] = useState({ loading: true, data: null, error: null })

  const run = useCallback(() => {
    let cancelled = false
    setState((s) => ({ ...s, loading: true, error: null }))

    fn()
      .then((data) => !cancelled && setState({ loading: false, data, error: null }))
      .catch((error) => !cancelled && setState({ loading: false, data: null, error }))

    return () => {
      cancelled = true
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  useEffect(run, [run])
  return { ...state, reload: run }
}

export function since(iso) {
  const secs = Math.max(1, (Date.now() - new Date(iso).getTime()) / 1000)
  const steps = [
    [60, 's'],
    [60, 'm'],
    [24, 'h'],
    [7, 'd'],
    [4.35, 'w'],
    [12, 'mo'],
  ]

  let value = secs
  let unit = 's'
  for (const [size, next] of steps) {
    if (value < size) break
    value /= size
    unit = next
  }
  return `${Math.floor(value)}${unit}`
}
