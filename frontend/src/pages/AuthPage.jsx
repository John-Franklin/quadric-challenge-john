import { useEffect, useRef, useState } from 'react'
import { Link as RouterLink, Navigate, useLocation, useNavigate } from 'react-router-dom'
import { Alert, Box, Button, Link, Stack, TextField, Typography } from '@mui/material'
import { useAuth } from '../auth/AuthContext'
import PageHeading from '../components/PageHeading'

const MIN_PASSWORD_LENGTH = 8

// Login and registration share one form; `mode` is 'login' or 'register'.
function AuthPage({ mode }) {
  const isRegister = mode === 'register'
  const { user, login, register } = useAuth()
  const navigate = useNavigate()
  const location = useLocation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [fieldErrors, setFieldErrors] = useState({})
  const [submitError, setSubmitError] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const emailRef = useRef(null)
  const passwordRef = useRef(null)
  const submitErrorRef = useRef(null)

  // The disabled submit button loses focus, so move it to the error that explains why.
  useEffect(() => {
    if (submitError) submitErrorRef.current?.focus()
  }, [submitError])

  const redirectTo = location.state?.from?.pathname || '/jobs'
  if (user) return <Navigate to={redirectTo} replace />

  function validate() {
    const errors = {}
    if (!/^[^\s@]+@[^\s@]+$/.test(email.trim())) errors.email = 'Please enter a valid email.'
    if (isRegister && password.length < MIN_PASSWORD_LENGTH) {
      errors.password = `Password must be at least ${MIN_PASSWORD_LENGTH} characters.`
    } else if (!password) {
      errors.password = 'Please enter your password.'
    }
    return errors
  }

  async function handleSubmit(event) {
    event.preventDefault()
    setSubmitError('')
    const errors = validate()
    setFieldErrors(errors)
    if (Object.keys(errors).length > 0) {
      (errors.email ? emailRef : passwordRef).current.focus()
      return
    }

    setSubmitting(true)
    try {
      await (isRegister ? register : login)(email.trim(), password)
      navigate(redirectTo, { replace: true })
    } catch (err) {
      setSubmitError(err.message)
      setSubmitting(false)
    }
  }

  return (
    <>
      <PageHeading>{isRegister ? 'Create Account' : 'Log In'}</PageHeading>
      <Box component="form" onSubmit={handleSubmit} noValidate sx={{ maxWidth: 400 }}>
        <Stack spacing={3}>
          {submitError && <Alert severity="error" ref={submitErrorRef} tabIndex={-1}>{submitError}</Alert>}
          <TextField
            id="email"
            label="Email"
            type="email"
            autoComplete="email"
            inputRef={emailRef}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            error={Boolean(fieldErrors.email)}
            helperText={fieldErrors.email}
            required
            disabled={submitting}
            fullWidth
          />
          <TextField
            id="password"
            label="Password"
            type="password"
            autoComplete={isRegister ? 'new-password' : 'current-password'}
            inputRef={passwordRef}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            error={Boolean(fieldErrors.password)}
            helperText={fieldErrors.password || (isRegister && `At least ${MIN_PASSWORD_LENGTH} characters`)}
            required
            disabled={submitting}
            fullWidth
          />
          <Box>
            <Button type="submit" variant="contained" loading={submitting}>
              {isRegister ? 'Create Account' : 'Log In'}
            </Button>
          </Box>
          <Typography sx={{ fontSize: 14 }}>
            {isRegister ? 'Already have an account? ' : "Don't have an account? "}
            <Link component={RouterLink} to={isRegister ? '/login' : '/register'} state={location.state}>
              {isRegister ? 'Log in' : 'Create one'}
            </Link>
          </Typography>
        </Stack>
      </Box>
    </>
  )
}

export default AuthPage
