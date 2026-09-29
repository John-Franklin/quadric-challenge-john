import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { Alert, Box, Button, MenuItem, Stack, TextField } from '@mui/material'
import api from '../api'
import PageHeading from '../components/PageHeading'
import { ACTIONS, CODE_ACTION } from '../jobActions'

// Per-action count inputs; limits must match the backend (see backend/handlers/job.go).
const COUNT_FIELDS = {
  calculate_pi: { name: 'digits', label: 'Digits', defaultValue: 1000, max: 10000, help: 'Decimal places of pi to calculate' },
  lorem_ipsum: { name: 'words', label: 'Words', defaultValue: 200, max: 5000, help: 'Number of words to generate' },
}

function CreateJob() {
  const navigate = useNavigate()
  const [action, setAction] = useState('')
  const [notes, setNotes] = useState('')
  const [code, setCode] = useState('')
  const [counts, setCounts] = useState({})
  const [countError, setCountError] = useState('')
  const [validationError, setValidationError] = useState('')
  const [codeError, setCodeError] = useState('')
  const [submitError, setSubmitError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const actionRef = useRef(null)
  const countRef = useRef(null)
  const codeRef = useRef(null)
  const submitErrorRef = useRef(null)
  const countField = COUNT_FIELDS[action]
  const countValue = counts[action] ?? String(countField?.defaultValue)

  // The disabled submit button loses focus, so move it to the error that explains why.
  useEffect(() => {
    if (submitError) submitErrorRef.current?.focus()
  }, [submitError])

  async function handleSubmit(event) {
    event.preventDefault()
    setSubmitError('')

    if (!ACTIONS.some((a) => a.value === action)) {
      setValidationError('Please select a valid action.')
      actionRef.current.focus()
      return
    }
    setValidationError('')

    const needsCode = action === CODE_ACTION
    if (needsCode && !code.trim()) {
      setCodeError('Please enter the Python code to run.')
      codeRef.current.focus()
      return
    }
    setCodeError('')

    let countParam = {}
    if (countField) {
      const count = Number(countValue)
      if (!/^\d+$/.test(countValue.trim()) || count < 1 || count > countField.max) {
        setCountError(`Please enter a whole number between 1 and ${countField.max}.`)
        countRef.current.focus()
        return
      }
      countParam = { [countField.name]: count }
    }
    setCountError('')

    setSubmitting(true)
    try {
      await api.jobs.create({ action, notes: notes.trim(), ...(needsCode && { code }), ...countParam })
      navigate('/jobs')
    } catch (err) {
      setSubmitError(`Failed to create job: ${err.message}`)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <>
      <PageHeading>Create Job</PageHeading>
      <Box
        component="form"
        id="job-form"
        onSubmit={handleSubmit}
        noValidate
        sx={{ maxWidth: 600 }}
      >
        <Stack spacing={3}>
          {submitError && <Alert severity="error" ref={submitErrorRef} tabIndex={-1}>{submitError}</Alert>}

          <TextField
            select
            id="action"
            inputRef={actionRef}
            label="Action"
            value={action}
            onChange={(e) => {
              setAction(e.target.value)
              setValidationError('')
              setCountError('')
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

          {countField && (
            <TextField
              id="count"
              inputRef={countRef}
              type="number"
              label={countField.label}
              value={countValue}
              onChange={(e) => {
                setCounts({ ...counts, [action]: e.target.value })
                setCountError('')
              }}
              error={Boolean(countError)}
              helperText={countError || `${countField.help} (1-${countField.max})`}
              required
              disabled={submitting}
              fullWidth
              slotProps={{ htmlInput: { min: 1, max: countField.max, step: 1 } }}
            />
          )}

          {action === CODE_ACTION && (
            <TextField
              id="code"
              inputRef={codeRef}
              label="Python Code"
              value={code}
              onChange={(e) => {
                setCode(e.target.value)
                setCodeError('')
              }}
              placeholder={"print('Hello from the runner')"}
              error={Boolean(codeError)}
              helperText={codeError || 'Executed with python3 on the runner; stdout and stderr appear in the job logs. Automatically terminates after 5 minutes to handle infinite loops.'}
              required
              multiline
              minRows={8}
              disabled={submitting}
              fullWidth
              slotProps={{ htmlInput: { spellCheck: false, sx: { fontFamily: 'monospace' } } }}
            />
          )}

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
