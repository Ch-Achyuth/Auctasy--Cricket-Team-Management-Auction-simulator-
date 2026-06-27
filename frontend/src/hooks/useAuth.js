import { useAuthContext } from '../context/AuthContext'

/**
 * useAuth — session state and auth actions, backed by the app-wide AuthContext.
 * The actual session subscription lives in <AuthProvider> so it runs once.
 */
export function useAuth() {
  const { session, user, authLoading, signInWithGoogle, signOut } = useAuthContext()
  return {
    session,
    user,
    loading: authLoading,
    signInWithGoogle,
    signOut,
  }
}
