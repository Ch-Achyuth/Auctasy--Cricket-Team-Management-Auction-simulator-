import { createClient } from '@supabase/supabase-js'

// Both values come from the root .env (Vite reads VITE_* via envDir: '../').
const supabaseUrl = import.meta.env.VITE_SUPABASE_URL
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY

// Fail loud at startup rather than booting a broken client with a placeholder
// key — a misconfigured deploy should be obvious, not silently half-working.
if (!supabaseUrl || !supabaseAnonKey) {
  throw new Error(
    'Supabase config missing. Set VITE_SUPABASE_URL and VITE_SUPABASE_ANON_KEY ' +
    'in the root .env (Vite reads them via envDir: "../").',
  )
}

export const supabase = createClient(supabaseUrl, supabaseAnonKey)
