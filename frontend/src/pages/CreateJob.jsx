import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Alert, Box, Button, MenuItem, Stack, TextField, Typography } from '@mui/material'

// Must match the actions supported by the runner (see /runner/README.md).
const ACTIONS = [
  { value: 'calculate_pi', label: 'Calculate Pi' },
  { value: 'lorem_ipsum', label: 'Lorem Ipsum' },
]

function CreateJob() {
  const navigate = useNavigate()
  const [action, setAction] = useState('')
  const [notes, setNotes] = useState('')
  const [validationError, setValidationError] = useState('')
  const [submitError, setSubmitError] = useState('')
  const [submitting, setSubmitting] = useState(false)

  async function handleSubmit(event) {
    event.preventDefault()
    setSubmitError('')

    if (!ACTIONS.some((a) => a.value === action)) {
      setValidationError('Please select a valid action.')
      return
    }
    setValidationError('')

    setSubmitting(true)
    try {
      const response = await fetch('/api/jobs', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action, notes: notes.trim() }),
      })
      if (!response.ok) {
        const body = await response.json().catch(() => null)
        throw new Error(body?.error || `Request failed with status ${response.status}`)
      }
      navigate('/jobs')
    } catch (err) {
      setSubmitError(`Failed to create job: ${err.message}`)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <Typography variant="h1">Create Job</Typography>
      <Box
        component="form"
        id="job-form"
        onSubmit={handleSubmit}
        noValidate
        sx={{ maxWidth: 600 }}
      >
        <Stack spacing={3}>
          {submitError && <Alert severity="error">{submitError}</Alert>}

          <TextField
            select
            id="action"
            label="Action"
            value={action}
            onChange={(e) => {
              setAction(e.target.value)
              setValidationError('')
            }}
            error={Boolean(validationError)}
            helperText={validationError || 'Choose the task to execute'}
            required
            disabled={submitting}
            fullWidth
          >
            {ACTIONS.map((a) => (
              <MenuItem key={a.value} value={a.value}>
                {a.label}
              </MenuItem>
            ))}
          </TextField>

          <TextField
            id="notes"
            label="Notes (Optional)"
            value={notes}
            onChange={(e) => setNotes(e.target.value)}
            placeholder="Add any notes or description for this job..."
            helperText="Optional description or context for this job"
            multiline
            rows={6}
            disabled={submitting}
            fullWidth
          />

          <Stack direction="row" spacing={2}>
            <Button type="submit" variant="contained" loading={submitting}>
              Create Job
            </Button>
            <Button
              variant="contained"
              color="secondary"
              onClick={() => navigate('/jobs')}
              disabled={submitting}
            >
              Cancel
            </Button>
          </Stack>
        </Stack>
      </Box>
    </>
  )
}

export default CreateJob
