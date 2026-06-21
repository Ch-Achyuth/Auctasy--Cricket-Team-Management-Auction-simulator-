-- ============================================================
-- Auctasy — User Profile Migration (ALTER TABLE)
-- Run this in your Supabase SQL Editor
-- ============================================================

-- 1. Drop the firebase_uid column (no longer needed with Supabase Auth)
ALTER TABLE public.users DROP COLUMN IF EXISTS firebase_uid;

-- 2. Add bio column
ALTER TABLE public.users ADD COLUMN IF NOT EXISTS bio TEXT;

-- 3. Make username NOT NULL (if it isn't already)
-- Note: This will fail if there are rows with NULL usernames.
-- Clean those up first if needed.
ALTER TABLE public.users ALTER COLUMN username SET NOT NULL;

-- 4. Make display_name NOT NULL
ALTER TABLE public.users ALTER COLUMN display_name SET NOT NULL;

-- 5. Remove the auto-generated UUID default from id
-- (id should now be explicitly set to match auth.users.id)
ALTER TABLE public.users ALTER COLUMN id DROP DEFAULT;

-- 6. Add foreign key constraint to auth.users
-- This links the users table to Supabase Auth
ALTER TABLE public.users
  ADD CONSTRAINT users_id_fkey
  FOREIGN KEY (id) REFERENCES auth.users(id) ON DELETE CASCADE;

-- ============================================================
-- 7. Enable Row Level Security
-- ============================================================
ALTER TABLE public.users ENABLE ROW LEVEL SECURITY;

-- ============================================================
-- 8. RLS Policies — all use auth.uid() for security
-- ============================================================

-- Users can read their own profile
CREATE POLICY "Users can view own profile"
  ON public.users
  FOR SELECT
  USING (auth.uid() = id);

-- Users can insert their own profile (only once, during onboarding)
CREATE POLICY "Users can insert own profile"
  ON public.users
  FOR INSERT
  WITH CHECK (auth.uid() = id);

-- Users can update their own profile
CREATE POLICY "Users can update own profile"
  ON public.users
  FOR UPDATE
  USING (auth.uid() = id)
  WITH CHECK (auth.uid() = id);

-- ============================================================
-- 9. Auto-update trigger for updated_at
-- ============================================================
CREATE OR REPLACE FUNCTION public.handle_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER on_users_updated
  BEFORE UPDATE ON public.users
  FOR EACH ROW
  EXECUTE FUNCTION public.handle_updated_at();
