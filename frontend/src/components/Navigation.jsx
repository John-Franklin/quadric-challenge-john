import { NavLink } from 'react-router-dom'

function Navigation() {
  return (
    <nav className="nav">
      <NavLink to="/jobs" end>Jobs</NavLink>
      <NavLink to="/create">Create Job</NavLink>
    </nav>
  )
}

export default Navigation
