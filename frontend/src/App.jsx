import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { useAuth } from './hooks/useAuth'
import Landing from './pages/Landing'
import Dashboard from './pages/Dashboard'
import Onboarding from './pages/Onboarding'
import Profile from './pages/Profile'
import NotFound from './pages/NotFound'
import ProtectedRoute from './components/ProtectedRoute'
import LoadingScreen from './components/LoadingScreen'

/**
 * App — root component with routing.
 * - /            → Landing (unauthenticated) or redirect to /dashboard
 * - /onboarding  → First-login profile setup (protected, skips profile check)
 * - /dashboard   → Main dashboard (protected, requires profile)
 * - /profile     → Edit profile (protected, requires profile)
 */
export default function App() {
  const { user, loading } = useAuth()

  // Show loading screen while checking auth (prevents flash)
  if (loading) {
    return <LoadingScreen />
  }

  return (
    <BrowserRouter>
      <Routes>
        {/* Landing — redirect to dashboard if already logged in */}
        <Route
          path="/"
          element={user ? <Navigate to="/dashboard" replace /> : <Landing />}
        />

        {/* Onboarding — protected but skips profile check (user hasn't created one yet) */}
        <Route
          path="/onboarding"
          element={
            <ProtectedRoute skipProfileCheck>
              <Onboarding />
            </ProtectedRoute>
          }
        />

        {/* Dashboard — protected, requires profile */}
        <Route
          path="/dashboard"
          element={
            <ProtectedRoute>
              <Dashboard />
            </ProtectedRoute>
          }
        />

        {/* Profile — protected, requires profile */}
        <Route
          path="/profile"
          element={
            <ProtectedRoute>
              <Profile />
            </ProtectedRoute>
          }
        />

        {/* Unmatched routes → 404 */}
        <Route path="*" element={<NotFound />} />
      </Routes>
    </BrowserRouter>
  )
}
