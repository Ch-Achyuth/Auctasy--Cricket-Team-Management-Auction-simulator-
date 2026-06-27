import { useState } from 'react'
import { useNavigate, Navigate } from 'react-router-dom'
import { useAuth } from '../hooks/useAuth'
import { useProfile } from '../hooks/useProfile'
import { useToast } from '../components/Toast'

/**
 * Onboarding page — shown on first login when no profile exists.
 * Collects username and display name, validates, then creates profile.
 */
export default function Onboarding() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const { profile, createProfile, checkUsernameAvailable } = useProfile()
  const { showToast, ToastContainer } = useToast()

  // Prefill display name from Google account
  const [displayName, setDisplayName] = useState(
    user?.user_metadata?.full_name || ''
  )
  const [username, setUsername] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [errors, setErrors] = useState({})

  // A user who already has a profile must never see (or re-submit) onboarding —
  // re-inserting would collide with their existing row. Declared after all hooks
  // so the early return never changes hook order.
  if (profile) return <Navigate to="/dashboard" replace />

  // Username validation: 3-20 chars, letters/numbers/underscores only
  const validateUsername = (value) => {
    if (value.length < 3) return 'Username must be at least 3 characters'
    if (value.length > 20) return 'Username must be 20 characters or less'
    if (!/^[a-zA-Z0-9_]+$/.test(value)) return 'Only letters, numbers, and underscores allowed'
    return null
  }

  const validateForm = () => {
    const newErrors = {}
    const usernameError = validateUsername(username)
    if (usernameError) newErrors.username = usernameError
    if (!displayName.trim()) newErrors.displayName = 'Display name is required'
    setErrors(newErrors)
    return Object.keys(newErrors).length === 0
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!validateForm()) return

    setSubmitting(true)
    try {
      // Check username availability
      const available = await checkUsernameAvailable(username)
      if (!available) {
        setErrors({ username: 'This username is already taken' })
        setSubmitting(false)
        return
      }

      await createProfile({
        username: username.toLowerCase(),
        displayName: displayName.trim(),
      })

      showToast('Profile created successfully!', 'success')
      setTimeout(() => navigate('/dashboard', { replace: true }), 500)
    } catch (err) {
      console.error('Onboarding error:', err)
      showToast(err.message || 'Failed to create profile', 'error')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="onboarding-container">
      <ToastContainer />

      <div className="onboarding-card">
        <img
          src={user?.user_metadata?.avatar_url || `https://ui-avatars.com/api/?name=${encodeURIComponent(displayName || 'U')}&background=0f172a&color=fff&size=128`}
          alt="Avatar"
          className="onboarding-avatar"
        />
        <h1 className="onboarding-title">Welcome to Auctasy</h1>
        <p className="onboarding-subtitle">Set up your profile to get started</p>

        <form onSubmit={handleSubmit} className="onboarding-form">
          <div className="form-group">
            <label htmlFor="username" className="form-label">Username</label>
            <input
              id="username"
              type="text"
              className={`input-field ${errors.username ? 'input-error' : ''}`}
              placeholder="e.g. cricket_king"
              value={username}
              onChange={(e) => {
                setUsername(e.target.value)
                setErrors((prev) => ({ ...prev, username: null }))
              }}
              maxLength={20}
              autoComplete="off"
            />
            {errors.username && <p className="field-error">{errors.username}</p>}
          </div>

          <div className="form-group">
            <label htmlFor="displayName" className="form-label">Display Name</label>
            <input
              id="displayName"
              type="text"
              className={`input-field ${errors.displayName ? 'input-error' : ''}`}
              placeholder="Your display name"
              value={displayName}
              onChange={(e) => {
                setDisplayName(e.target.value)
                setErrors((prev) => ({ ...prev, displayName: null }))
              }}
              autoComplete="off"
            />
            {errors.displayName && <p className="field-error">{errors.displayName}</p>}
          </div>

          <button
            type="submit"
            className="submit-button"
            disabled={submitting}
          >
            {submitting ? 'Creating...' : 'Create Profile'}
          </button>
        </form>
      </div>
    </div>
  )
}
