import { useState, useEffect } from 'react'
import { supabase } from '../lib/supabase'

/**
 * useAuth hook — manages Supabase auth session state.
 * Provides session, user, loading state, and auth actions.
 */
export function useAuth() {
  const [session, setSession] = useState(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    // Get initial session
    supabase.auth.getSession().then(({ data: { session } }) => {
      setSession(session)
      setLoading(false)

      // Clear access tokens from URL after OAuth callback
      if (session && (window.location.hash.includes('access_token') || window.location.search.includes('code='))) {
        window.history.replaceState({}, document.title, window.location.pathname)
      }
    })

    // Listen for auth state changes (login, logout, token refresh)
    const { data: { subscription } } = supabase.auth.onAuthStateChange((_event, session) => {
      setSession(session)
    })

    return () => subscription.unsubscribe()
  }, [])

  const signInWithGoogle = async () => {
    await supabase.auth.signInWithOAuth({
      provider: 'google',
      options: {
        redirectTo: window.location.origin,
      },
    })
  }

  const signOut = async () => {
    await supabase.auth.signOut()
  }

  return {
    session,
    user: session?.user ?? null,
    loading,
    signInWithGoogle,
    signOut,
  }
}
