import { Navigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { useProfile } from '../hooks/useProfile'

/**
 * ProtectedRoute — route guard component.
 * 1. If not authenticated → redirect to /
 * 2. If authenticated but no profile → redirect to /onboarding
 *    (unless we're already on the onboarding page)
 * 3. Otherwise → render children
 */
export default function ProtectedRoute({ children, skipProfileCheck = false }) {
  const { user, loading: authLoading } = useAuth()
  const { profile, loading: profileLoading } = useProfile(user)

  // Show loading state while checking auth + profile
  if (authLoading || profileLoading) {
    return (
      <div className="premium-center">
        <h1 className="project-title">Auctasy</h1>
        <p className="subtitle" style={{ opacity: 0.5 }}>Loading...</p>
      </div>
    )
  }

  // Not logged in → send to landing
  if (!user) {
    return <Navigate to="/" replace />
  }

  // Logged in but no profile → send to onboarding
  // (skip this check on the onboarding page itself)
  if (!skipProfileCheck && !profile) {
    return <Navigate to="/onboarding" replace />
  }

  return children
}
