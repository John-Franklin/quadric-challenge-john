import { useEffect, useState } from 'react'
import { IconButton, Tooltip } from '@mui/material'
import CheckIcon from '@mui/icons-material/Check'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'

function CopyButton({ text, label }) {
  const [status, setStatus] = useState('idle')

  useEffect(() => {
    if (status === 'idle') return
    const timer = setTimeout(() => setStatus('idle'), 2000)
    return () => clearTimeout(timer)
  }, [status])

  async function handleCopy() {
    try {
      await navigator.clipboard.writeText(text)
      setStatus('copied')
    } catch {
      setStatus('failed')
    }
  }

  const title = status === 'copied' ? 'Copied!' : status === 'failed' ? 'Copy failed' : label

  return (
    <>
      <Tooltip title={title}>
        <IconButton size="small" onClick={handleCopy} aria-label={label}>
          {status === 'copied' ? (
            <CheckIcon fontSize="small" color="success" />
          ) : (
            <ContentCopyIcon fontSize="small" />
          )}
        </IconButton>
      </Tooltip>
      <span role="status" className="visually-hidden">{status === 'idle' ? '' : title}</span>
    </>
  )
}

export default CopyButton
