import { useState } from 'react'
import { NavLink, useNavigate } from 'react-router-dom'
import { Button, Chip } from '@mui/material'
import { useAuth } from '../auth/AuthContext'

function Navigation() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [loggingOut, setLoggingOut] = useState(false)

  async function handleLogout() {
    setLoggingOut(true)
    try {
      await logout()
      navigate('/login')
    } finally {
      setLoggingOut(false)
    }
  }

  return (
    <nav className="nav" aria-label="Main">
      {user ? (
        <>
          <NavLink to="/jobs" end>Jobs</NavLink>
          <NavLink to="/create">Create Job</NavLink>
          {user.is_admin && <NavLink to="/admin/users">Users</NavLink>}
          <div className="nav-account">
            <span className="nav-email">
              <span className="visually-hidden">Signed in as </span>
              {user.email}
            </span>
            {user.is_admin && <Chip size="small" label="Admin" color="secondary" />}
            <Button size="small" onClick={handleLogout} loading={loggingOut}>
              Log out
            </Button>
          </div>
        </>
      ) : (
        user === null && (
          <>
            <NavLink to="/login">Log In</NavLink>
            <NavLink to="/register">Create Account</NavLink>
          </>
        )
      )}
    </nav>
  )
}

export default Navigation
