import { useState, useEffect, useCallback } from 'react'
import { supabase } from '../lib/supabase'

/**
 * useProfile hook — manages the user's profile in public.users.
 * Fetches profile on mount, provides CRUD operations.
 */
export function useProfile(user) {
  const [profile, setProfile] = useState(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState(null)

  // Fetch profile for the current user
  const fetchProfile = useCallback(async () => {
    if (!user) {
      setProfile(null)
      setLoading(false)
      return
    }

    try {
      setLoading(true)
      setError(null)

      const { data, error: fetchError } = await supabase
        .from('users')
        .select('*')
        .eq('id', user.id)
        .single()

      if (fetchError && fetchError.code !== 'PGRST116') {
        // PGRST116 = "no rows returned" — that just means no profile yet
        throw fetchError
      }

      setProfile(data || null)
    } catch (err) {
      console.error('Error fetching profile:', err)
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }, [user])

  useEffect(() => {
    fetchProfile()
  }, [fetchProfile])

  /**
   * Create a new profile during onboarding.
   * Inserts a row into public.users linked to auth.users via RLS.
   */
  const createProfile = async ({ username, displayName }) => {
    if (!user) throw new Error('Not authenticated')

    const { data, error: insertError } = await supabase
      .from('users')
      .insert({
        id: user.id,
        username,
        display_name: displayName,
        email: user.email,
        avatar_url: user.user_metadata?.avatar_url || null,
      })
      .select()
      .single()

    if (insertError) throw insertError

    setProfile(data)
    return data
  }

  /**
   * Update editable profile fields (display_name, bio).
   */
  const updateProfile = async ({ displayName, bio }) => {
    if (!user) throw new Error('Not authenticated')

    const { data, error: updateError } = await supabase
      .from('users')
      .update({
        display_name: displayName,
        bio,
      })
      .eq('id', user.id)
      .select()
      .single()

    if (updateError) throw updateError

    setProfile(data)
    return data
  }

  /**
   * Check if a username is available (case-insensitive).
   * Returns true if the username is free to use.
   */
  const checkUsernameAvailable = async (username) => {
    const { data, error: checkError } = await supabase
      .from('users')
      .select('id')
      .ilike('username', username)
      .maybeSingle()

    if (checkError) throw checkError

    return data === null // null means no match — username is available
  }

  return {
    profile,
    loading,
    error,
    fetchProfile,
    createProfile,
    updateProfile,
    checkUsernameAvailable,
  }
}
