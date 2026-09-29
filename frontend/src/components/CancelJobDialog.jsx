import { useRef, useState } from 'react'
import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
} from '@mui/material'
import { crumpleIntoTrash } from '../utils/crumpleIntoTrash'

// Confirms cancelling `job`; open while `job` is non-null. `onConfirm` returns a promise.
function CancelJobDialog({ job, onClose, onConfirm }) {
  const [submitting, setSubmitting] = useState(false)
  const [error, setError] = useState('')
  const [tossing, setTossing] = useState(false)
  const paperRef = useRef(null)
  // Keeps the last job rendered during the close transition, after `job` becomes null.
  const [shown, setShown] = useState(job)
  if (job && job !== shown) setShown(job)

  function handleClose() {
    if (submitting) return
    setError('')
    onClose()
  }

  async function handleConfirm() {
    setSubmitting(true)
    setError('')
    try {
      await onConfirm(job.id)
    } catch (err) {
      setError(`Failed to cancel job: ${err.message}`)
      setSubmitting(false)
      return
    }
    setTossing(true)
    await crumpleIntoTrash(paperRef.current)
    onClose()
    setTossing(false)
    setSubmitting(false)
  }

  return (
    <Dialog
      open={Boolean(job)}
      onClose={handleClose}
      aria-labelledby="cancel-job-title"
      slotProps={{ paper: { ref: paperRef } }}
    >
      <DialogTitle id="cancel-job-title">Cancel job #{shown?.id}?</DialogTitle>
      <DialogContent>
        {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
        <DialogContentText>
          {shown?.status === 'running'
            ? `The ${shown.action} job is running. The runner will stop it at its next log update.`
            : `The ${shown?.action} job will be removed from the queue before it starts.`}
        </DialogContentText>
      </DialogContent>
      <DialogActions sx={{ px: 3, pb: 2 }}>
        <Button variant="contained" color="secondary" onClick={handleClose} disabled={submitting}>
          Keep Job
        </Button>
        <Button variant="contained" color="error" onClick={handleConfirm} loading={submitting && !tossing}>
          Cancel Job
        </Button>
      </DialogActions>
    </Dialog>
  )
}

export default CancelJobDialog
