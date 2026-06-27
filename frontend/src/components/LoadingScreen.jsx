/**
 * LoadingScreen — the single full-page loading state, reused by the app root
 * and route guards so the loading UI is consistent everywhere.
 */
export default function LoadingScreen({ message = 'Loading…' }) {
  return (
    <div className="premium-center">
      <h1 className="project-title">Auctasy</h1>
      <p className="subtitle" style={{ opacity: 0.5 }}>{message}</p>
    </div>
  )
}
