import { Link } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { useProfile } from '../hooks/useProfile'
import ProfileCard from '../components/ProfileCard'

/**
 * Dashboard — main page after login.
 * Shows a welcome, the profile card, and navigation.
 */
export default function Dashboard() {
  const { signOut } = useAuth()
  const { profile } = useProfile()

  const firstName = (profile?.display_name || '').split(' ')[0]

  return (
    <div className="premium-center">
      <h1 className="project-title">Auctasy</h1>
      <p className="subtitle">
        {firstName ? `Welcome back, ${firstName}.` : 'Welcome back.'}
      </p>

      {profile && <ProfileCard profile={profile} />}

      <div className="dashboard-actions">
        <Link to="/profile" className="submit-button" style={{ textDecoration: 'none' }}>
          Edit Profile
        </Link>
        <button className="secondary-button" onClick={signOut}>
          Logout
        </button>
      </div>
    </div>
  )
}
