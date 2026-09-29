import { Chip } from '@mui/material'

// Badge colors from styles.txt.
const STATUS_COLORS = {
  pending: { bgcolor: '#fff3cd', color: '#856404' },
  running: { bgcolor: '#cfe2ff', color: '#084298' },
  completed: { bgcolor: '#d1e7dd', color: '#0f5132' },
  failed: { bgcolor: '#f8d7da', color: '#842029' },
}

function StatusBadge({ status }) {
  const normalized = (status || 'unknown').toLowerCase()
  const label = normalized.charAt(0).toUpperCase() + normalized.slice(1)
  return (
    <Chip
      label={label}
      size="small"
      sx={{ fontSize: 12, fontWeight: 600, ...STATUS_COLORS[normalized] }}
    />
  )
}

export default StatusBadge
