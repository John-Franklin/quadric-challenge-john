import { Box, CircularProgress, Typography } from '@mui/material'

function LoadingIndicator({ children }) {
  return (
    <Box role="status" sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
      <CircularProgress size={20} aria-hidden="true" />
      <Typography color="text.secondary">{children}</Typography>
    </Box>
  )
}

export default LoadingIndicator
