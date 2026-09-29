import { BrowserRouter, Route, Navigate } from 'react-router-dom'
import AnimatedRoutes from './components/AnimatedRoutes'
import Navigation from './components/Navigation'
import RequireAuth from './components/RequireAuth'
import AuthProvider from './auth/AuthProvider'
import JobsList from './pages/JobsList'
import CreateJob from './pages/CreateJob'
import AuthPage from './pages/AuthPage'
import AdminUsers from './pages/AdminUsers'

function App() {
  return (
    <BrowserRouter>
      <AuthProvider>
        <a className="skip-link" href="#main-content">Skip to main content</a>
        <div className="container">
          <Navigation />
          <main id="main-content" className="content" tabIndex={-1}>
            <AnimatedRoutes>
              <Route path="/" element={<Navigate to="/jobs" replace />} />
              <Route path="/login" element={<AuthPage mode="login" />} />
              <Route path="/register" element={<AuthPage mode="register" />} />
              <Route path="/jobs" element={<RequireAuth><JobsList /></RequireAuth>} />
              <Route path="/create" element={<RequireAuth><CreateJob /></RequireAuth>} />
              <Route path="/admin/users" element={<RequireAuth admin><AdminUsers /></RequireAuth>} />
            </AnimatedRoutes>
          </main>
        </div>
      </AuthProvider>
    </BrowserRouter>
  )
}

export default App
