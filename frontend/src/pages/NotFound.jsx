import { Link } from 'react-router-dom'

/**
 * NotFound — 404 page for unmatched routes.
 */
export default function NotFound() {
  return (
    <div className="premium-center">
      <h1 className="project-title">404</h1>
      <p className="subtitle">This page doesn’t exist.</p>
      <Link to="/" className="submit-button" style={{ textDecoration: 'none' }}>
        Back to home
      </Link>
    </div>
  )
}
