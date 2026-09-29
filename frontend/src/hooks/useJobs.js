import { useEffect, useState } from 'react'

function useJobs() {
  const [jobs, setJobs] = useState([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    const controller = new AbortController()

    async function loadJobs() {
      try {
        const response = await fetch('/api/jobs', { signal: controller.signal })
        if (!response.ok) {
          const body = await response.json().catch(() => null)
          throw new Error(body?.error || `Request failed with status ${response.status}`)
        }
        setJobs(await response.json())
      } catch (err) {
        if (err.name !== 'AbortError') setError(`Failed to load jobs: ${err.message}`)
      } finally {
        if (!controller.signal.aborted) setLoading(false)
      }
    }

    loadJobs()
    return () => controller.abort()
  }, [])

  return { jobs, loading, error }
}

export default useJobs
