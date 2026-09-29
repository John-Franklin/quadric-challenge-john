import { useCallback, useEffect, useRef, useState } from 'react'
import api from '../api'

const POLL_INTERVAL_MS = 7000
// Finished jobs never change, so only rows in these statuses need refreshing.
const ACTIVE_STATUSES = ['pending', 'running']
// Matches the backend's per-request cap for GET /api/jobs?ids=.
const IDS_PER_REQUEST = 100

// Refetches loaded rows below the newest page that are still active.
async function fetchActiveOlderJobs(state, newestPage, signal) {
  if (!state) return []
  const inNewest = new Set(newestPage.jobs.map((job) => job.id))
  const ids = state.jobs
    .filter((job) => ACTIVE_STATUSES.includes(job.status) && !inNewest.has(job.id))
    .map((job) => job.id)

  const batches = []
  for (let i = 0; i < ids.length; i += IDS_PER_REQUEST) {
    batches.push(api.jobs.listByIds(ids.slice(i, i + IDS_PER_REQUEST), { signal }))
  }
  return (await Promise.all(batches)).flatMap((result) => result.jobs)
}

function applyUpdates(state, updated) {
  if (updated.length === 0) return state
  const byId = new Map(updated.map((job) => [job.id, job]))
  return { ...state, jobs: state.jobs.map((job) => byId.get(job.id) ?? job) }
}

function fromPage(page) {
  return { jobs: page.jobs, hasMore: page.has_more, nextBefore: page.next_before, total: page.total }
}

// Merges a freshly fetched newest page into the loaded list: rows it contains are
// updated in place and newer jobs are prepended.
function mergeNewest(state, page) {
  if (!state || state.jobs.length === 0) return fromPage(page)

  const newestLoaded = state.jobs[0].id
  const oldestFetched = page.jobs.at(-1)?.id ?? 0
  // More than a page of jobs arrived since the last poll, so the list would have a gap.
  if (page.has_more && oldestFetched > newestLoaded) return fromPage(page)

  const fresh = new Map(page.jobs.map((job) => [job.id, job]))
  const added = page.jobs.filter((job) => job.id > newestLoaded)
  return {
    ...state,
    jobs: [...added, ...state.jobs.map((job) => fresh.get(job.id) ?? job)],
    total: page.total,
  }
}

// Loads jobs newest first for infinite scrolling: `loadMore` appends the next page. In
// the background it polls the newest page for new jobs and status changes, and refetches
// any still-active rows further down so their status stays current too.
function useJobs() {
  const [state, setState] = useState(null)
  const [error, setError] = useState('')
  const [loadingMore, setLoadingMore] = useState(false)
  const [loadMoreError, setLoadMoreError] = useState('')
  const loadingMoreRef = useRef(false)
  // Lets the polling effect, which runs once, read the latest loaded rows.
  const stateRef = useRef(state)
  stateRef.current = state

  useEffect(() => {
    const controller = new AbortController()
    let timer

    async function refresh() {
      try {
        const page = await api.jobs.list(null, { signal: controller.signal })
        const olderUpdates = await fetchActiveOlderJobs(stateRef.current, page, controller.signal)
        setState((current) => applyUpdates(mergeNewest(current, page), olderUpdates))
        setError('')
      } catch (err) {
        if (err.name !== 'AbortError') setError(`Failed to load jobs: ${err.message}`)
      } finally {
        // Chained rather than setInterval so a slow request never overlaps the next one.
        if (!controller.signal.aborted) timer = setTimeout(poll, POLL_INTERVAL_MS)
      }
    }

    function poll() {
      if (document.hidden) {
        timer = setTimeout(poll, POLL_INTERVAL_MS)
        return
      }
      refresh()
    }

    refresh()
    return () => {
      controller.abort()
      clearTimeout(timer)
    }
  }, [])

  const hasMore = state?.hasMore ?? false
  const nextBefore = state?.nextBefore

  const loadMore = useCallback(async () => {
    if (loadingMoreRef.current || !hasMore) return
    loadingMoreRef.current = true
    setLoadingMore(true)
    setLoadMoreError('')
    try {
      const page = await api.jobs.list(nextBefore)
      setState((current) => {
        // A poll reset the list while this page was loading; appending would leave a gap.
        if (current.nextBefore !== nextBefore) return current
        const loaded = new Set(current.jobs.map((job) => job.id))
        return {
          ...current,
          jobs: [...current.jobs, ...page.jobs.filter((job) => !loaded.has(job.id))],
          hasMore: page.has_more,
          nextBefore: page.next_before,
          total: page.total,
        }
      })
    } catch (err) {
      setLoadMoreError(`Failed to load more jobs: ${err.message}`)
    } finally {
      loadingMoreRef.current = false
      setLoadingMore(false)
    }
  }, [hasMore, nextBefore])

  // Throws on failure so the caller can show the error where the action was taken.
  const cancelJob = useCallback(async (id) => {
    const body = await api.jobs.cancel(id)
    setState((current) => ({
      ...current,
      jobs: current.jobs.map((job) => (job.id === body.id ? body : job)),
    }))
  }, [])

  return {
    jobs: state?.jobs ?? [],
    total: state?.total ?? 0,
    loaded: state !== null,
    hasMore,
    error,
    loadingMore,
    loadMoreError,
    loadMore,
    cancelJob,
  }
}

export default useJobs
