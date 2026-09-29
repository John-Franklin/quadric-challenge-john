import { useEffect, useState } from 'react'
import {
  Alert,
  Switch,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
} from '@mui/material'
import api from '../api'
import { useAuth } from '../auth/AuthContext'
import LoadingIndicator from '../components/LoadingIndicator'
import PageHeading from '../components/PageHeading'
import { formatTimestamp } from '../utils/formatTimestamp'

function AdminUsers() {
  const { user: currentUser } = useAuth()
  const [users, setUsers] = useState(null)
  const [error, setError] = useState('')
  const [savingId, setSavingId] = useState(null)
  const [announcement, setAnnouncement] = useState('')

  useEffect(() => {
    const controller = new AbortController()
    api.admin.listUsers({ signal: controller.signal })
      .then(setUsers)
      .catch((err) => {
        if (err.name !== 'AbortError') setError(`Failed to load users: ${err.message}`)
      })
    return () => controller.abort()
  }, [])

  // Controls stay enabled while saving (disabling them would drop keyboard focus);
  // changes made mid-save are ignored instead.
  async function updateUser(id, changes) {
    if (savingId !== null) return
    setSavingId(id)
    setError('')
    setAnnouncement('')
    try {
      const updated = await api.admin.updateUser(id, changes)
      setUsers((current) => current.map((u) => (u.id === updated.id ? updated : u)))
      setAnnouncement(`Saved changes for ${updated.email}.`)
    } catch (err) {
      setError(`Failed to update user: ${err.message}`)
    } finally {
      setSavingId(null)
    }
  }

  return (
    <>
      <PageHeading>Users</PageHeading>
      <span role="status" className="visually-hidden">{announcement}</span>
      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {!users ? (
        !error && <LoadingIndicator>Loading users...</LoadingIndicator>
      ) : (
        <TableContainer>
          <Table aria-label="Users">
            <TableHead>
              <TableRow>
                <TableCell>ID</TableCell>
                <TableCell>Email</TableCell>
                <TableCell>Joined</TableCell>
                <TableCell>Admin</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {users.map((u) => {
                const isSelf = u.id === currentUser.id
                return (
                  <TableRow key={u.id} hover>
                    <TableCell>#{u.id}</TableCell>
                    <TableCell component="th" scope="row">
                      {u.email}
                      {isSelf && <span className="visually-hidden"> (you)</span>}
                    </TableCell>
                    <TableCell sx={{ fontSize: 13, whiteSpace: 'nowrap' }}>
                      <time dateTime={u.created_at}>{formatTimestamp(u.created_at)}</time>
                    </TableCell>
                    <TableCell>
                      <Switch
                        checked={u.is_admin}
                        onChange={(e) => updateUser(u.id, { is_admin: e.target.checked })}
                        // Mirrors the backend rule: admins cannot demote themselves.
                        disabled={isSelf}
                        slotProps={{
                          input: {
                            'aria-label': `Admin access for ${u.email}`,
                            'aria-describedby': isSelf ? 'self-admin-note' : undefined,
                          },
                        }}
                      />
                    </TableCell>
                  </TableRow>
                )
              })}
            </TableBody>
          </Table>
        </TableContainer>
      )}
      <span id="self-admin-note" className="visually-hidden">You can't remove your own admin access.</span>
    </>
  )
}

export default AdminUsers
