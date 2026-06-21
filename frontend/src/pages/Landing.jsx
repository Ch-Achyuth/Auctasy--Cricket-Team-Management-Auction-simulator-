import { useAuth } from '../hooks/useAuth'
import WaitingListCounter from '../components/WaitingListCounter'

/**
 * Landing page — shown to unauthenticated users.
 * Preserves the original "Auctasy / Under Development" design.
 */
export default function Landing() {
  const { signInWithGoogle } = useAuth()

  return (
    <div className="premium-center">
      <h1 className="project-title">Auctasy</h1>
      <p className="subtitle">Under Development</p>

      <div className="auth-section">
        <button className="google-login-btn" onClick={signInWithGoogle}>
          <img
            src="https://upload.wikimedia.org/wikipedia/commons/5/53/Google_%22G%22_Logo.svg"
            alt="Google Logo"
            className="google-icon"
          />
          Login with Google
        </button>
      </div>

      <WaitingListCounter />
    </div>
  )
}
