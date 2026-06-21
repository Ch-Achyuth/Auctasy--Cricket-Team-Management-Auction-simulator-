import { createClient } from '@supabase/supabase-js'

// Reads from VITE_ prefixed env vars in frontend/.env
const supabaseUrl = import.meta.env.VITE_SUPABASE_URL || 'https://joxkejcuzqhkboowueba.supabase.co'
const supabaseAnonKey = import.meta.env.VITE_SUPABASE_ANON_KEY

if (!supabaseAnonKey) {
  console.error(
    '⚠️ VITE_SUPABASE_ANON_KEY is missing!\n' +
    'Create a file at frontend/.env with:\n' +
    'VITE_SUPABASE_URL=https://joxkejcuzqhkboowueba.supabase.co\n' +
    'VITE_SUPABASE_ANON_KEY=your-anon-key-here'
  )
}

// Use a placeholder key to prevent crash — auth calls will fail gracefully
export const supabase = createClient(
  supabaseUrl,
  supabaseAnonKey || 'missing-key-check-console'
)
