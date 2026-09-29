import { useEffect, useRef, useState } from 'react'

// Calls `onVisible` whenever this element is on screen and `disabled` is false. It fires
// again each time `onVisible` changes while still visible, so short pages keep loading
// until the viewport is filled.
function InfiniteScrollTrigger({ onVisible, disabled, rootMargin = '100px' }) {
  const ref = useRef(null)
  const [visible, setVisible] = useState(false)

  useEffect(() => {
    const observer = new IntersectionObserver(([entry]) => setVisible(entry.isIntersecting), {
      rootMargin,
    })
    observer.observe(ref.current)
    return () => observer.disconnect()
  }, [rootMargin])

  useEffect(() => {
    if (visible && !disabled) onVisible()
  }, [visible, disabled, onVisible])

  return <div ref={ref} aria-hidden="true" />
}

export default InfiniteScrollTrigger
