import { Component } from 'react'

/**
 * ErrorBoundary — catches render-time errors anywhere below it and shows a
 * recoverable fallback instead of a blank white screen.
 */
export default class ErrorBoundary extends Component {
  constructor(props) {
    super(props)
    this.state = { hasError: false, error: null }
  }

  static getDerivedStateFromError(error) {
    return { hasError: true, error }
  }

  componentDidCatch(error, info) {
    console.error('Uncaught render error:', error, info)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="premium-center">
          <h1 className="project-title">Auctasy</h1>
          <p className="subtitle">Something went wrong.</p>
          <button
            className="submit-button"
            onClick={() => window.location.assign('/')}
          >
            Reload app
          </button>
        </div>
      )
    }
    return this.props.children
  }
}
