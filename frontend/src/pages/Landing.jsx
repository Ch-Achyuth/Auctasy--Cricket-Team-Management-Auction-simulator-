import { useAuth } from '../hooks/useAuth'

/**
 * Landing page — shown to unauthenticated users.
 */
export default function Landing() {
  const { signInWithGoogle } = useAuth()

  return (
    <div className="premium-center">
      <h1 className="project-title">Auctasy</h1>
      <p className="subtitle">Build your dream squad. Bid live. Win your league.</p>

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
    </div>
  )
}
