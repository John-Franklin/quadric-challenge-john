import { useState } from 'react'
import { Routes, useLocation } from 'react-router-dom'

// Routes in nav order; moving to a later one slides content left, an earlier one right.
const ROUTE_ORDER = ['/jobs', '/create', '/admin/users']

// Like <Routes>, but slides between pages. The outgoing page stays mounted until its
// exit animation ends; both are keyed by pathname so React keeps the same component
// instance (and its state) when a page switches from current to exiting.
function AnimatedRoutes({ children }) {
  const location = useLocation()
  const [current, setCurrent] = useState(location)
  const [exiting, setExiting] = useState(null)
  const [direction, setDirection] = useState(null)

  if (location !== current) {
    const from = ROUTE_ORDER.indexOf(current.pathname)
    const to = ROUTE_ORDER.indexOf(location.pathname)
    // Query-only changes (e.g. ?page=) and redirects from unlisted paths don't animate.
    if (location.pathname !== current.pathname && from !== -1 && to !== -1) {
      setExiting(current)
      setDirection(to > from ? 'left' : 'right')
    }
    setCurrent(location)
  }

  const pages = exiting ? [exiting, current] : [current]

  return (
    <div className="page-transition">
      {pages.map((pageLocation) => {
        const isExiting = pageLocation === exiting
        const animation = direction && `page-${isExiting ? 'exit' : 'enter'}-${direction}`
        return (
          <div
            key={pageLocation.pathname}
            className={['page', isExiting && 'page-exiting', animation].filter(Boolean).join(' ')}
            inert={isExiting}
            onAnimationEnd={(event) => {
              // Ignore animations bubbling up from inside the page (e.g. spinners).
              if (isExiting && event.target === event.currentTarget) setExiting(null)
            }}
          >
            <Routes location={pageLocation}>{children}</Routes>
          </div>
        )
      })}
    </div>
  )
}

export default AnimatedRoutes
