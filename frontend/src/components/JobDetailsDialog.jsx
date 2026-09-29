import { useEffect, useLayoutEffect, useRef, useState } from 'react'
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  Typography,
} from '@mui/material'
import api from '../api'
import { CODE_ACTION } from '../jobActions'
import CopyButton from './CopyButton'
import LoadingIndicator from './LoadingIndicator'
import StatusBadge from './StatusBadge'

const ACTIVE_STATUSES = ['pending', 'running']
const POLL_INTERVAL_MS = 1000

const blockSx = {
  m: 0,
  p: 2,
  overflow: 'auto',
  bgcolor: '#f6f8fa',
  borderRadius: 1,
  fontSize: 13,
  whiteSpace: 'pre-wrap',
  wordBreak: 'break-word',
}

function Section({ title, copyText, children }) {
  return (
    <Box component="section">
      <Stack direction="row" sx={{ alignItems: 'center', justifyContent: 'space-between', mb: 0.5 }}>
        <Typography variant="subtitle2" component="h3">{title}</Typography>
        {copyText && <CopyButton text={copyText} label={`Copy ${title.toLowerCase()}`} />}
      </Stack>
      {children}
    </Box>
  )
}

// Shows the code (for Python jobs) and logs of `job`; open while `job` is non-null.
function JobDetailsDialog({ job, onClose, canCancel, onCancel }) {
  const [details, setDetails] = useState(null)
  const [error, setError] = useState('')
  // Announces status transitions (e.g. running -> completed) that are otherwise only visual.
  const [statusMessage, setStatusMessage] = useState('')
  // Keeps the last job rendered during the close transition, after `job` becomes null.
  const [shown, setShown] = useState(job)
  const [prevJob, setPrevJob] = useState(job)
  if (job !== prevJob) {
    setPrevJob(job)
    if (job) {
      setShown(job)
      setDetails(null)
      setError('')
      setStatusMessage('')
    }
  }
  // Keep the log view pinned to the newest line unless the user has scrolled up.
  const logsRef = useRef(null)
  const pinnedToBottom = useRef(true)
  const lastStatus = useRef(null)

  // Refetches while the job is still active so logs stream in; stops once it is finished.
  useEffect(() => {
    if (!job) return
    const controller = new AbortController()
    let timer
    pinnedToBottom.current = true
    lastStatus.current = null

    async function load() {
      let active = true
      try {
        const body = await api.jobs.get(job.id, { signal: controller.signal })
        setDetails(body)
        setError('')
        if (lastStatus.current && lastStatus.current !== body.status) {
          setStatusMessage(`Job #${job.id} is now ${body.status}.`)
        }
        lastStatus.current = body.status
        active = ACTIVE_STATUSES.includes(body.status)
      } catch (err) {
        if (err.name === 'AbortError') return
        setError(`Failed to load job details: ${err.message}`)
      }
      if (active && !controller.signal.aborted) timer = setTimeout(poll, POLL_INTERVAL_MS)
    }

    function poll() {
      if (document.hidden) {
        timer = setTimeout(poll, POLL_INTERVAL_MS)
        return
      }
      load()
    }

    load()
    return () => {
      controller.abort()
      clearTimeout(timer)
    }
  }, [job])

  useLayoutEffect(() => {
    const el = logsRef.current
    if (el && pinnedToBottom.current) el.scrollTop = el.scrollHeight
  }, [details?.logs])

  function handleLogsScroll(event) {
    const el = event.currentTarget
    pinnedToBottom.current = el.scrollHeight - el.scrollTop - el.clientHeight < 16
  }

  const status = details?.status ?? shown?.status

  return (
    <Dialog open={Boolean(job)} onClose={onClose} aria-labelledby="job-details-title" maxWidth="md" fullWidth>
      <DialogTitle id="job-details-title" sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
        Job #{shown?.id}
        <Typography component="span" color="text.secondary">{shown?.action}</Typography>
        <StatusBadge status={status} />
      </DialogTitle>
      <DialogContent>
        <span role="status" className="visually-hidden">{statusMessage}</span>
        {/* A failed refresh keeps the last loaded details visible below the error. */}
        {error && <Alert severity="error" sx={{ mb: details ? 2 : 0 }}>{error}</Alert>}
        {!details ? (
          !error && <LoadingIndicator>Loading job details...</LoadingIndicator>
        ) : (
          <Stack spacing={3}>
            {(details.digits || details.words) && (
              <Typography color="text.secondary">
                {details.digits
                  ? `Digits: ${details.digits.toLocaleString()}`
                  : `Words: ${details.words.toLocaleString()}`}
              </Typography>
            )}
            {details.action === CODE_ACTION && (
              <Section title="Code" copyText={details.code}>
                <Box component="pre" tabIndex={0} role="region" aria-label="Code" sx={{ ...blockSx, maxHeight: '30vh' }}>
                  {details.code}
                </Box>
              </Section>
            )}
            <Section title="Logs" copyText={details.logs}>
              {details.logs ? (
                <Box
                  component="pre"
                  ref={logsRef}
                  onScroll={handleLogsScroll}
                  tabIndex={0}
                  role="region"
                  aria-label="Logs"
                  sx={{ ...blockSx, maxHeight: '50vh' }}
                >
                  {details.logs}
                </Box>
              ) : (
                <Typography color="text.secondary">No logs yet.</Typography>
              )}
            </Section>
          </Stack>
        )}
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2 }}>
        {canCancel && ACTIVE_STATUSES.includes(status) && (
          <Button
            variant="outlined"
            color="error"
            onClick={() => onCancel(shown)}
            aria-label={`Cancel job #${shown.id}`}
          >
            Cancel job
          </Button>
        )}
        <Button variant="contained" color="secondary" onClick={onClose}>
          Close
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default JobDetailsDialog
