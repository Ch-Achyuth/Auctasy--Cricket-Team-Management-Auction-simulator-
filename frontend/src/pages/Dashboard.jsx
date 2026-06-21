import { Link } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { useProfile } from '../hooks/useProfile'
import ProfileCard from '../components/ProfileCard'
import WaitingListCounter from '../components/WaitingListCounter'

/**
 * Dashboard — main page after login.
 * Shows profile card, navigation links, and the waiting list counter.
 */
export default function Dashboard() {
  const { user, signOut } = useAuth()
  const { profile } = useProfile(user)

  return (
    <div className="premium-center">
      <h1 className="project-title">Auctasy</h1>
      <p className="subtitle">Under Development</p>

      {profile && <ProfileCard profile={profile} />}

      <div className="dashboard-actions">
        <Link to="/profile" className="submit-button" style={{ textDecoration: 'none' }}>
          Edit Profile
        </Link>
        <button className="secondary-button" onClick={signOut}>
          Logout
        </button>
      </div>

      <WaitingListCounter />
    </div>
  )
}
