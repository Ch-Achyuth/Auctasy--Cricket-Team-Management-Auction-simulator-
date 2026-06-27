import { createContext, useContext, useState, useEffect, useCallback } from 'react'
import { supabase } from '../lib/supabase'

/**
 * AuthContext — single source of truth for the Supabase session AND the
 * user's profile row. Mounted once at the app root so session subscriptions
 * and the profile fetch happen exactly once, instead of being duplicated by
 * every component that calls useAuth()/useProfile().
 */
const AuthContext = createContext(null)

export function AuthProvider({ children }) {
  const [session, setSession] = useState(null)
  const [authLoading, setAuthLoading] = useState(true)
  const [profile, setProfile] = useState(null)
  const [profileLoading, setProfileLoading] = useState(true)
  const [profileError, setProfileError] = useState(null)

  // ── Session lifecycle ────────────────────────────────────────────────────
  useEffect(() => {
    supabase.auth.getSession().then(({ data: { session } }) => {
      setSession(session)
      setAuthLoading(false)

      // Strip OAuth artifacts (tokens, ?code=, ?error=) from the URL after the
      // callback so a refresh doesn't re-trigger or leak them.
      const { hash, search } = window.location
      if (
        hash.includes('access_token') ||
        search.includes('code=') ||
        search.includes('error=')
      ) {
        window.history.replaceState({}, document.title, window.location.pathname)
      }
    })

    const { data: { subscription } } = supabase.auth.onAuthStateChange((_event, session) => {
      setSession(session)
    })

    return () => subscription.unsubscribe()
  }, [])

  const user = session?.user ?? null
  const userId = user?.id ?? null

  // ── Profile fetch (re-runs when the signed-in user changes) ───────────────
  const fetchProfile = useCallback(async () => {
    if (!userId) {
      setProfile(null)
      setProfileLoading(false)
      return
    }

    try {
      setProfileLoading(true)
      setProfileError(null)

      // maybeSingle: returns null (not an error) when the row doesn't exist yet.
      const { data, error } = await supabase
        .from('users')
        .select('*')
        .eq('id', userId)
        .maybeSingle()

      if (error) throw error
      setProfile(data ?? null)
    } catch (err) {
      console.error('Error fetching profile:', err)
      setProfileError(err.message)
      setProfile(null)
    } finally {
      setProfileLoading(false)
    }
  }, [userId])

  useEffect(() => {
    fetchProfile()
  }, [fetchProfile])

  // ── Auth actions ──────────────────────────────────────────────────────────
  const signInWithGoogle = useCallback(async () => {
    await supabase.auth.signInWithOAuth({
      provider: 'google',
      options: { redirectTo: window.location.origin },
    })
  }, [])

  const signOut = useCallback(async () => {
    await supabase.auth.signOut()
  }, [])

  // ── Profile actions ─────────────────────────────────────────────────────
  /**
   * Create (or re-affirm) the signed-in user's profile. Uses upsert so a stale
   * "no profile" UI state can never collide with an existing row — the operation
   * is idempotent on the primary key. A username already taken by *another* user
   * surfaces as a friendly error rather than a raw Postgres constraint message.
   */
  const createProfile = useCallback(async ({ username, displayName }) => {
    if (!user) throw new Error('Not authenticated')

    const { data, error } = await supabase
      .from('users')
      .upsert(
        {
          id: user.id,
          username,
          display_name: displayName,
          email: user.email,
          avatar_url: user.user_metadata?.avatar_url || null,
        },
        { onConflict: 'id' },
      )
      .select()
      .single()

    if (error) {
      if (error.code === '23505') throw new Error('That username is already taken')
      throw error
    }

    setProfile(data)
    return data
  }, [user])

  const updateProfile = useCallback(async ({ displayName, bio }) => {
    if (!user) throw new Error('Not authenticated')

    const { data, error } = await supabase
      .from('users')
      .update({ display_name: displayName, bio })
      .eq('id', user.id)
      .select()
      .single()

    if (error) throw error
    setProfile(data)
    return data
  }, [user])

  /**
   * Best-effort availability check against the public_profiles view (readable
   * regardless of RLS). The DB unique constraint + the 23505 handling in
   * createProfile remain the authoritative guard against the check→insert race.
   */
  const checkUsernameAvailable = useCallback(async (username) => {
    const { data, error } = await supabase
      .from('public_profiles')
      .select('id')
      .ilike('username', username)
      .limit(1)

    if (error) throw error
    return !data || data.length === 0
  }, [])

  const value = {
    session,
    user,
    authLoading,
    profile,
    profileLoading,
    profileError,
    fetchProfile,
    signInWithGoogle,
    signOut,
    createProfile,
    updateProfile,
    checkUsernameAvailable,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuthContext() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuthContext must be used within an <AuthProvider>')
  return ctx
}
