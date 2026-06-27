import { useAuthContext } from '../context/AuthContext'

/**
 * useProfile — the current user's profile and profile actions, backed by the
 * app-wide AuthContext. The profile is fetched once in <AuthProvider>, so this
 * no longer triggers a duplicate fetch per component.
 *
 * The optional `user` argument is accepted for backward compatibility with
 * existing call sites and is ignored — the provider owns the user identity.
 */
export function useProfile() {
  const {
    profile,
    profileLoading,
    profileError,
    fetchProfile,
    createProfile,
    updateProfile,
    checkUsernameAvailable,
  } = useAuthContext()

  return {
    profile,
    loading: profileLoading,
    error: profileError,
    fetchProfile,
    createProfile,
    updateProfile,
    checkUsernameAvailable,
  }
}
