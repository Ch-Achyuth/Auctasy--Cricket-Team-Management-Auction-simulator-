/**
 * ProfileCard — reusable component that displays a user's profile.
 * Used in both the Profile and Dashboard pages.
 */
export default function ProfileCard({ profile }) {
  if (!profile) return null

  return (
    <div className="profile-card">
      <img
        src={profile.avatar_url || `https://ui-avatars.com/api/?name=${encodeURIComponent(profile.display_name)}&background=0f172a&color=fff&size=128`}
        alt={profile.display_name}
        className="profile-avatar"
      />
      <div className="profile-info">
        <h2 className="profile-display-name">{profile.display_name}</h2>
        <p className="profile-username">@{profile.username}</p>
        {profile.bio && <p className="profile-bio">{profile.bio}</p>}
        <p className="profile-email">{profile.email}</p>
      </div>
    </div>
  )
}
