import { Navigate, useLocation } from 'react-router-dom'
import { Box, CircularProgress } from '@mui/material'
import { useAuth } from '../auth/AuthContext'

// Renders children only for a logged-in user (and only admins when `admin` is set).
function RequireAuth({ admin = false, children }) {
  const { user } = useAuth()
  const location = useLocation()

  if (user === undefined) {
    return (
      <Box role="status" sx={{ display: 'flex', justifyContent: 'center', mt: 4 }}>
        <CircularProgress size={24} aria-hidden="true" />
        <span className="visually-hidden">Checking session...</span>
      </Box>
    )
  }
  if (!user) return <Navigate to="/login" replace state={{ from: location }} />
  if (admin && !user.is_admin) return <Navigate to="/jobs" replace />
  return children
}

export default RequireAuth
