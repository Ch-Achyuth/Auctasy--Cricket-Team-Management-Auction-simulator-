# CLAUDE.md

Cricket team management auction simulator. Private leagues, live IPL player auctions, squad building within a budget, match-day lineups, fantasy points from real performances.

## Stack

- **Frontend**: React 19, Vite (port 8000), Supabase JS client, plain JSX (no TypeScript)
- **Backend**: Go (`github.com/ch-achyuth/auctasy`), `net/http`, `gorilla/websocket`, `redis/go-redis/v9`, `supabase-community/supabase-go`
- **Database**: Supabase Postgres — 16 tables, RLS active
- **Cache/Pub-Sub**: Upstash Redis

## Architecture

```
Browser  ──Supabase JS──▶  Supabase (auth + public reads)
Browser  ──POST /api/v1/auction/bid──▶  Go server ──place_bid RPC──▶ Postgres
Browser  ──WebSocket /ws──▶  Go Hub ◀──▶ Redis Pub/Sub (auction_room:{league_id})
```

Go holds the `service_role` key. Browser uses `anon` key. All financial mutations (bids, budget deductions, ownership transfers) go exclusively through Go → SECURITY DEFINER stored procedures. The anon key has zero direct write access to any transactional table.

## Repo layout

```
.env                             All secrets — root level, shared by Go and Vite
supabase.sql                     DB schema reference (do not edit; use migration files)
migrations/
  002_rls_and_rpcs.sql           RLS + stored procedures  [APPLIED]

backend/
  cmd/server/main.go             HTTP server — health check + bid endpoint
  cmd/seed/main.go               CSV seeder → player_historical_stats
  internal/config/config.go      Env loader (SUPABASE_URL, SUPABASE_SERVICE_KEY,
                                  SUPABASE_JWT_SECRET, REDIS_URL, PORT)
  internal/database/
    database.go                  Supabase client + CallRPC(ctx, fn, jsonParams []byte)
    redis.go                     Redis client wrapper
  internal/middleware/auth.go    HS256 JWT validation → injects user UUID (CtxUserID) into ctx
  internal/handlers/
    health.go                    GET /health
    bid.go                       POST /api/v1/auction/bid → calls place_bid RPC
  internal/auction/
    bid_validator.go             Pure bid validation, no DB calls — ValidateBid(BidRequest)
    bid_validator_test.go        13 tests
  internal/scoring/
    engine.go                    IPL fantasy points — Calculate(MatchPerformance) Points
    engine_test.go               19 tests
  internal/websocket/
    hub.go                       Hub: register/unregister/broadcast channels + sync.RWMutex
                                  Redis Pub/Sub per league, Publish(ctx, leagueID, event)
    client.go                    Client: readPump (discard-only) + writePump + ping/pong

frontend/
  vite.config.js                 envDir: '../' — reads VITE_* from root .env
  src/lib/supabase.js            Supabase client (VITE_SUPABASE_URL, VITE_SUPABASE_ANON_KEY)
  src/hooks/useAuth.js           Session, signIn, signOut
  src/hooks/useProfile.js        Profile CRUD, username check
  src/pages/                     Landing, Onboarding, Dashboard, Profile
  src/components/                ProtectedRoute, ProfileCard, Toast, WaitingListCounter
```

## Environment (root `.env`)

```
SUPABASE_URL=
SUPABASE_SERVICE_KEY=      # Go only — never sent to browser
SUPABASE_JWT_SECRET=       # Go JWT middleware (Supabase dashboard → Settings → API)
REDIS_URL=
VITE_SUPABASE_URL=         # Vite picks this up via envDir: '../'
VITE_SUPABASE_ANON_KEY=
PORT=8080                  # optional, defaults to 8080
```

## Commands

```bash
# from backend/
go run cmd/server/main.go          # HTTP + WS server
go test ./internal/...             # 32 unit tests
go build ./...                     # compile check

# from frontend/
npm run dev                        # Vite on port 8000
npm run build
```

## Stored procedures (live in Supabase)

| RPC | Caller | Does atomically |
|-----|--------|-----------------|
| `place_bid(p_user_id, p_auction_queue_id, p_league_id, p_bid_amount)` | Go bid handler | Validates membership → locks queue row → checks bid vs base/high/budget → inserts bid → updates winning_bid → appends WAL event |
| `settle_auction_item(p_auction_queue_id)` | Go auction clock | Marks sold/unsold → deducts budget → records ownership → writes transaction → appends event |
| `get_missed_events(p_auction_id, p_after_seq)` | Frontend on reconnect | Returns all `auction_events` with `seq > p_after_seq` for state reconciliation |

`auction_events` has a `seq BIGSERIAL` column. Reconnecting clients pass their last seen `seq` to catch up on missed events before resuming the live WebSocket stream.

## Current state

**Working:**
- Frontend: Google OAuth → onboarding → profile, routing guards, toasts
- Go backend: `GET /health`, `POST /api/v1/auction/bid` (JWT-protected)
- WebSocket hub written and compiled; Redis Pub/Sub fan-out per league
- RLS active on all 19 tables; 3 stored procedures live
- 32 unit tests passing (bid validator + scoring engine)
- Root `.env` is the single secrets file; Vite configured with `envDir: '../'`

**Not yet built:**
- `/ws` route not wired into `cmd/server/main.go`
- League CRUD (create, join via invite code, list)
- Auction session lifecycle (start/pause/end, player queue, auction clock goroutine)
- Frontend auction room (bid button, countdown, live WebSocket state)
- Lineup selection UI
- Scoring pipeline (match + player_performances → fantasy points per user)
- Leaderboard, trades, chat, achievements, deployment

**Next tasks (in order):**
1. Wire `/ws` into the server: authenticate with `RequireAuth`, extract `leagueID` from query param, call `websocket.ServeWS(hub, w, r, userID, leagueID)`; start `hub.Run(ctx)` as a goroutine on startup
2. League CRUD API + frontend — unblocks all auction work
3. Auction session lifecycle + clock goroutine (calls `settle_auction_item` on timer expiry)

## Hard rules

- **No floats for money.** Budget/bid = `bigint` in Postgres, `int64` in Go, integers in JS.
- **WebSocket = broadcast only.** `hub.go` / `client.go` never touch the DB or call RPCs.
- **Financial mutations → Go → RPC.** Never add direct write policies on `bids`, `transactions`, `player_ownerships`, `auction_queue`, `league_users` for the `authenticated` role.
- **Schema changes → new migration file** (`migrations/00N_*.sql`). Never edit `supabase.sql`.
- **No Axios.** Frontend uses native `fetch` or the Supabase JS client.
- **Plain JSX.** No TypeScript. New hooks/components follow `useAuth`/`useProfile` pattern.
