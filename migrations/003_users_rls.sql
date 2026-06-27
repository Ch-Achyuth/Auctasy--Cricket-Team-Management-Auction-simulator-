-- ============================================================
-- Migration 003: RLS for public.users + safe public profile view
--
-- Migration 002 enabled RLS on 18 tables but omitted public.users
-- (the table predates the Firebase→Supabase auth migration — note the
-- leftover firebase_uid column). With RLS off, that was permissive; the
-- moment RLS is enabled in the dashboard with no policy, users can no
-- longer read their own profile row, so the app bounces them to
-- onboarding and the profile INSERT collides with the existing primary
-- key ("duplicate key value violates unique constraint users_pkey").
--
-- This migration makes the table's protection explicit and correct:
--   - a user may read / insert / update ONLY their own row
--   - email is never exposed to other users
--   - a separate view exposes non-sensitive columns for leaderboards
-- ============================================================

-- ─────────────────────────────────────────────────────────────
-- 1. Enable RLS and define self-scoped policies
-- ─────────────────────────────────────────────────────────────
ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS users_self_read   ON public.users;
DROP POLICY IF EXISTS users_self_insert ON public.users;
DROP POLICY IF EXISTS users_self_update ON public.users;

-- A user can read their own full profile row (includes email).
CREATE POLICY users_self_read
  ON public.users FOR SELECT
  USING (auth.uid() = id);

-- A user can create only their own row (id must equal their auth uid).
CREATE POLICY users_self_insert
  ON public.users FOR INSERT
  WITH CHECK (auth.uid() = id);

-- A user can update only their own row.
CREATE POLICY users_self_update
  ON public.users FOR UPDATE
  USING (auth.uid() = id)
  WITH CHECK (auth.uid() = id);

-- No DELETE policy: profiles are not user-deletable.

-- ─────────────────────────────────────────────────────────────
-- 2. Public profile view — safe columns only (NO email)
--
--    A view owned by a privileged role bypasses the underlying
--    table's RLS, so this exposes exactly these columns to everyone
--    (anon + authenticated) for leaderboards and username checks,
--    while email and firebase_uid stay private to the row owner.
-- ─────────────────────────────────────────────────────────────
CREATE OR REPLACE VIEW public.public_profiles AS
  SELECT id, username, display_name, avatar_url, total_leagues_won
  FROM public.users;

GRANT SELECT ON public.public_profiles TO anon, authenticated;
