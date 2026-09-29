const pad = (n) => String(n).padStart(2, '0')

// Renders as "YYYY-MM-DD HH:MM:SS" in the user's local time zone.
export function formatTimestamp(value) {
  const d = new Date(value)
  if (Number.isNaN(d.getTime())) return value
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ` +
    `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
  )
}
