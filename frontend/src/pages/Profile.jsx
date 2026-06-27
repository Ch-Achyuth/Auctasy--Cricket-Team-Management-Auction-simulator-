import { useState } from 'react'
import { useProfile } from '../hooks/useProfile'
import { useToast } from '../components/Toast'
import ProfileCard from '../components/ProfileCard'

/**
 * Profile page — displays user profile with editable fields.
 * Email and username are read-only. Display name and bio are editable.
 */
export default function Profile() {
  const { profile, updateProfile } = useProfile()
  const { showToast, ToastContainer } = useToast()

  const [editing, setEditing] = useState(false)
  const [displayName, setDisplayName] = useState('')
  const [bio, setBio] = useState('')
  const [saving, setSaving] = useState(false)

  const startEditing = () => {
    setDisplayName(profile?.display_name || '')
    setBio(profile?.bio || '')
    setEditing(true)
  }

  const cancelEditing = () => {
    setEditing(false)
  }

  const handleSave = async () => {
    if (!displayName.trim()) {
      showToast('Display name is required', 'error')
      return
    }

    setSaving(true)
    try {
      await updateProfile({
        displayName: displayName.trim(),
        bio: bio.trim(),
      })
      showToast('Profile updated!', 'success')
      setEditing(false)
    } catch (err) {
      console.error('Profile update error:', err)
      showToast(err.message || 'Failed to update profile', 'error')
    } finally {
      setSaving(false)
    }
  }

  if (!profile) return null

  return (
    <div className="profile-container">
      <ToastContainer />

      <h1 className="page-title">Your Profile</h1>

      {!editing ? (
        <>
          <ProfileCard profile={profile} />
          <button className="submit-button" onClick={startEditing}>
            Edit Profile
          </button>
        </>
      ) : (
        <div className="profile-edit-card">
          <img
            src={profile.avatar_url || `https://ui-avatars.com/api/?name=${encodeURIComponent(profile.display_name)}&background=0f172a&color=fff&size=128`}
            alt={profile.display_name}
            className="profile-avatar"
          />

          <div className="form-group">
            <label className="form-label">Username</label>
            <input
              type="text"
              className="input-field input-readonly"
              value={`@${profile.username}`}
              disabled
            />
          </div>

          <div className="form-group">
            <label className="form-label">Email</label>
            <input
              type="text"
              className="input-field input-readonly"
              value={profile.email}
              disabled
            />
          </div>

          <div className="form-group">
            <label htmlFor="edit-display-name" className="form-label">Display Name</label>
            <input
              id="edit-display-name"
              type="text"
              className="input-field"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
            />
          </div>

          <div className="form-group">
            <label htmlFor="edit-bio" className="form-label">Bio</label>
            <textarea
              id="edit-bio"
              className="input-field textarea-field"
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="Tell us about yourself..."
              rows={3}
            />
          </div>

          <div className="button-row">
            <button
              className="submit-button"
              onClick={handleSave}
              disabled={saving}
            >
              {saving ? 'Saving...' : 'Save Changes'}
            </button>
            <button
              className="secondary-button"
              onClick={cancelEditing}
              disabled={saving}
            >
              Cancel
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
