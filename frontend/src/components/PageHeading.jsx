import { useEffect, useRef } from 'react'
import { Typography } from '@mui/material'

let isInitialPage = true

// Page <h1> that sets the document title and, after client-side navigation, takes focus
// so screen readers announce the new page.
function PageHeading({ children }) {
  const ref = useRef(null)

  useEffect(() => {
    document.title = `${children} | Quadric Challenge`
    if (isInitialPage) {
      isInitialPage = false
      return
    }
    ref.current?.focus({ preventScroll: true })
  }, [children])

  return (
    <Typography variant="h1" ref={ref} tabIndex={-1}>
      {children}
    </Typography>
  )
}

export default PageHeading
