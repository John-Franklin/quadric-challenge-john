import { useCallback, useEffect, useMemo, useState } from 'react'
import api from '../api'
import { AuthContext } from './AuthContext'

// `user` is undefined while the session is being checked, null when logged out.
function AuthProvider({ children }) {
  const [user, setUser] = useState(undefined)

  useEffect(() => {
    const controller = new AbortController()
    api.auth.me({ signal: controller.signal })
      .then(setUser)
      .catch((err) => {
        if (err.name !== 'AbortError') setUser(null)
      })
    return () => controller.abort()
  }, [])

  const login = useCallback(async (email, password) => {
    setUser(await api.auth.login(email, password))
  }, [])

  const register = useCallback(async (email, password) => {
    setUser(await api.auth.register(email, password))
  }, [])

  const logout = useCallback(async () => {
    await api.auth.logout()
    setUser(null)
  }, [])

  const value = useMemo(() => ({ user, login, register, logout }), [user, login, register, logout])
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export default AuthProvider
