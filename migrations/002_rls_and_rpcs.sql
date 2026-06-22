-- ============================================================
-- Migration 002: RLS Policies + Stored Procedures
--
-- Security model:
--   anon key        → reads public/non-sensitive data only; zero direct writes to
--                     any transactional table
--   authenticated   → reads own scoped data; writes own non-financial rows
--                     (lineups, chat)
--   service_role    → bypasses RLS entirely (Go backend, seed scripts)
--   SECURITY DEFINER RPCs → only path for financial mutations (bids, budgets,
--                     ownership transfers); validate caller inside the function
-- ============================================================

-- ─────────────────────────────────────────────────────────────
-- 1. Add monotonic sequence to auction_events for ordered catch-up
--    Reconnecting clients pass their last-seen seq; the backend
--    streams only the events they missed.
-- ─────────────────────────────────────────────────────────────
ALTER TABLE public.auction_events
  ADD COLUMN IF NOT EXISTS seq BIGSERIAL NOT NULL;

CREATE INDEX IF NOT EXISTS auction_events_auction_seq_idx
  ON public.auction_events (auction_id, seq);

-- ─────────────────────────────────────────────────────────────
-- 2. Enable RLS on every table
-- ─────────────────────────────────────────────────────────────
ALTER TABLE public.teams                   ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.players                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.modes                   ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.leagues                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.league_users            ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.matches                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.player_performances     ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.auctions                ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.auction_queue           ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.bids                    ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.auction_events          ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.player_ownerships       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.lineups                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.trades                  ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.transactions            ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.chat_messages           ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.achievements            ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.user_achievements       ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.player_historical_stats ENABLE ROW LEVEL SECURITY;

-- ─────────────────────────────────────────────────────────────
-- 3. Public read-only reference tables (anon + authenticated)
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "teams_public_read"
  ON public.teams FOR SELECT USING (true);

CREATE POLICY "players_public_read"
  ON public.players FOR SELECT USING (true);

CREATE POLICY "modes_public_read"
  ON public.modes FOR SELECT USING (true);

CREATE POLICY "matches_public_read"
  ON public.matches FOR SELECT USING (true);

CREATE POLICY "player_performances_public_read"
  ON public.player_performances FOR SELECT USING (true);

CREATE POLICY "player_historical_stats_public_read"
  ON public.player_historical_stats FOR SELECT USING (true);

CREATE POLICY "achievements_public_read"
  ON public.achievements FOR SELECT USING (true);

-- ─────────────────────────────────────────────────────────────
-- 4. Leagues — public read; no direct client writes
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "leagues_public_read"
  ON public.leagues FOR SELECT USING (true);

-- ─────────────────────────────────────────────────────────────
-- 5. league_users — members see their own league's rows
--    No INSERT/UPDATE/DELETE: only SECURITY DEFINER RPCs write here
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "league_users_member_read"
  ON public.league_users FOR SELECT
  USING (
    auth.uid() IS NOT NULL AND
    league_id IN (
      SELECT lu.league_id FROM public.league_users lu
      WHERE lu.user_id = auth.uid()
    )
  );

-- ─────────────────────────────────────────────────────────────
-- 6. Auctions — league members can read
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "auctions_league_member_read"
  ON public.auctions FOR SELECT
  USING (
    auth.uid() IS NOT NULL AND
    league_id IN (
      SELECT lu.league_id FROM public.league_users lu
      WHERE lu.user_id = auth.uid()
    )
  );

-- ─────────────────────────────────────────────────────────────
-- 7. auction_queue — league members can read; no client writes
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "auction_queue_league_member_read"
  ON public.auction_queue FOR SELECT
  USING (
    auth.uid() IS NOT NULL AND
    auction_id IN (
      SELECT a.id FROM public.auctions a
      INNER JOIN public.league_users lu ON lu.league_id = a.league_id
      WHERE lu.user_id = auth.uid()
    )
  );

-- ─────────────────────────────────────────────────────────────
-- 8. bids — league members can read; no client writes
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "bids_league_member_read"
  ON public.bids FOR SELECT
  USING (
    auth.uid() IS NOT NULL AND
    auction_queue_id IN (
      SELECT aq.id FROM public.auction_queue aq
      INNER JOIN public.auctions a ON a.id = aq.auction_id
      INNER JOIN public.league_users lu ON lu.league_id = a.league_id
      WHERE lu.user_id = auth.uid()
    )
  );

-- ─────────────────────────────────────────────────────────────
-- 9. auction_events — league members can read; no client writes
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "auction_events_league_member_read"
  ON public.auction_events FOR SELECT
  USING (
    auth.uid() IS NOT NULL AND
    auction_id IN (
      SELECT a.id FROM public.auctions a
      INNER JOIN public.league_users lu ON lu.league_id = a.league_id
      WHERE lu.user_id = auth.uid()
    )
  );

-- ─────────────────────────────────────────────────────────────
-- 10. player_ownerships — league members can read; no client writes
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "player_ownerships_league_member_read"
  ON public.player_ownerships FOR SELECT
  USING (
    auth.uid() IS NOT NULL AND
    league_id IN (
      SELECT lu.league_id FROM public.league_users lu
      WHERE lu.user_id = auth.uid()
    )
  );

-- ─────────────────────────────────────────────────────────────
-- 11. transactions — users see their own rows only
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "transactions_own_read"
  ON public.transactions FOR SELECT USING (auth.uid() = user_id);

-- ─────────────────────────────────────────────────────────────
-- 12. lineups — users manage their own; no foreign user writes
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "lineups_own_read"
  ON public.lineups FOR SELECT USING (auth.uid() = user_id);

CREATE POLICY "lineups_own_insert"
  ON public.lineups FOR INSERT
  WITH CHECK (
    auth.uid() = user_id AND
    league_id IN (
      SELECT lu.league_id FROM public.league_users lu
      WHERE lu.user_id = auth.uid()
    )
  );

CREATE POLICY "lineups_own_update"
  ON public.lineups FOR UPDATE
  USING (auth.uid() = user_id)
  WITH CHECK (auth.uid() = user_id);

CREATE POLICY "lineups_own_delete"
  ON public.lineups FOR DELETE USING (auth.uid() = user_id);

-- ─────────────────────────────────────────────────────────────
-- 13. trades — only parties involved can read; no direct writes
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "trades_participant_read"
  ON public.trades FOR SELECT
  USING (auth.uid() = initiated_by OR auth.uid() = offered_to);

-- ─────────────────────────────────────────────────────────────
-- 14. chat_messages — league members can read and insert own rows
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "chat_messages_league_member_read"
  ON public.chat_messages FOR SELECT
  USING (
    auth.uid() IS NOT NULL AND
    league_id IN (
      SELECT lu.league_id FROM public.league_users lu
      WHERE lu.user_id = auth.uid()
    )
  );

CREATE POLICY "chat_messages_own_insert"
  ON public.chat_messages FOR INSERT
  WITH CHECK (
    auth.uid() = user_id AND
    league_id IN (
      SELECT lu.league_id FROM public.league_users lu
      WHERE lu.user_id = auth.uid()
    )
  );

-- ─────────────────────────────────────────────────────────────
-- 15. user_achievements — users read their own
-- ─────────────────────────────────────────────────────────────
CREATE POLICY "user_achievements_own_read"
  ON public.user_achievements FOR SELECT USING (auth.uid() = user_id);

-- ─────────────────────────────────────────────────────────────
-- 16. Stored Procedure: place_bid
--
--     Called from the Go backend (service_role key) after the JWT
--     middleware has authenticated the user. p_user_id is the
--     verified caller — it is NOT taken from auth.uid() because
--     service_role calls have no session context.
--
--     Atomically:
--       a) Verifies the caller is a league member
--       b) Locks the auction_queue row (prevents concurrent bid races)
--       c) Validates bid amount vs base price / current high / budget
--       d) Inserts the bid row
--       e) Updates auction_queue.winning_bid
--       f) Appends a bid_placed event to auction_events (the WAL)
-- ─────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION public.place_bid(
  p_user_id          uuid,
  p_auction_queue_id uuid,
  p_league_id        uuid,
  p_bid_amount       bigint
)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
  v_queue_status text;
  v_current_high bigint;
  v_base_price   bigint;
  v_auction_id   uuid;
  v_user_budget  bigint;
  v_bid_id       uuid;
  v_event_seq    bigint;
BEGIN
  -- Caller must be a member of the specified league
  IF NOT EXISTS (
    SELECT 1 FROM public.league_users
    WHERE league_id = p_league_id AND user_id = p_user_id
  ) THEN
    RETURN jsonb_build_object('success', false, 'error', 'not_league_member');
  END IF;

  -- Lock the auction queue row to prevent concurrent bid races
  SELECT aq.status, aq.winning_bid, aq.base_price, aq.auction_id
  INTO   v_queue_status, v_current_high, v_base_price, v_auction_id
  FROM   public.auction_queue aq
  WHERE  aq.id = p_auction_queue_id
  FOR UPDATE;

  IF NOT FOUND THEN
    RETURN jsonb_build_object('success', false, 'error', 'queue_item_not_found');
  END IF;

  -- Confirm the auction queue item belongs to the given league
  IF NOT EXISTS (
    SELECT 1 FROM public.auctions a
    WHERE a.id = v_auction_id AND a.league_id = p_league_id
  ) THEN
    RETURN jsonb_build_object('success', false, 'error', 'queue_league_mismatch');
  END IF;

  IF v_queue_status <> 'active' THEN
    RETURN jsonb_build_object('success', false, 'error', 'auction_not_active');
  END IF;

  -- Bid must meet or exceed base price (first bid) or exceed current high
  IF v_current_high IS NULL THEN
    IF p_bid_amount < v_base_price THEN
      RETURN jsonb_build_object('success', false, 'error', 'bid_below_base_price');
    END IF;
  ELSE
    IF p_bid_amount <= v_current_high THEN
      RETURN jsonb_build_object('success', false, 'error', 'bid_not_high_enough');
    END IF;
  END IF;

  -- Lock the bidder's budget row
  SELECT remaining_budget
  INTO   v_user_budget
  FROM   public.league_users
  WHERE  league_id = p_league_id AND user_id = p_user_id
  FOR UPDATE;

  IF v_user_budget < p_bid_amount THEN
    RETURN jsonb_build_object('success', false, 'error', 'insufficient_budget');
  END IF;

  -- Insert the bid record
  INSERT INTO public.bids (auction_queue_id, user_id, bid_amount)
  VALUES (p_auction_queue_id, p_user_id, p_bid_amount)
  RETURNING id INTO v_bid_id;

  -- Update the leading bid on the auction queue row
  UPDATE public.auction_queue
  SET    winning_bid = p_bid_amount, winning_user_id = p_user_id
  WHERE  id = p_auction_queue_id;

  -- Append to the write-ahead event log
  INSERT INTO public.auction_events (auction_id, event_type, payload_json)
  VALUES (
    v_auction_id,
    'bid_placed',
    jsonb_build_object(
      'bid_id',            v_bid_id,
      'user_id',           p_user_id,
      'bid_amount',        p_bid_amount,
      'auction_queue_id',  p_auction_queue_id
    )
  )
  RETURNING seq INTO v_event_seq;

  RETURN jsonb_build_object(
    'success',   true,
    'bid_id',    v_bid_id,
    'event_seq', v_event_seq
  );
END;
$$;

COMMENT ON FUNCTION public.place_bid IS
  'Atomically validates and records a bid. Called by the Go backend with the service_role key after JWT auth.';

-- ─────────────────────────────────────────────────────────────
-- 17. Stored Procedure: settle_auction_item
--
--     Triggered by the Go auction clock when the item timer expires.
--     Atomically:
--       a) Closes the queue item (sold / unsold)
--       b) Deducts the winning bid from the winner's remaining_budget
--       c) Inserts a player_ownerships row
--       d) Appends a transaction record
--       e) Logs the settlement event to auction_events
-- ─────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION public.settle_auction_item(
  p_auction_queue_id uuid
)
RETURNS jsonb
LANGUAGE plpgsql
SECURITY DEFINER
SET search_path = public
AS $$
DECLARE
  v_auction_id   uuid;
  v_league_id    uuid;
  v_player_id    uuid;
  v_winner_id    uuid;
  v_winning_bid  bigint;
  v_queue_status text;
  v_event_seq    bigint;
BEGIN
  -- Lock and read the queue row + its parent auction's league
  SELECT aq.auction_id, aq.player_id, aq.winning_user_id,
         aq.winning_bid, aq.status, a.league_id
  INTO   v_auction_id, v_player_id, v_winner_id,
         v_winning_bid, v_queue_status, v_league_id
  FROM   public.auction_queue aq
  INNER JOIN public.auctions a ON a.id = aq.auction_id
  WHERE  aq.id = p_auction_queue_id
  FOR UPDATE;

  IF NOT FOUND THEN
    RETURN jsonb_build_object('success', false, 'error', 'queue_item_not_found');
  END IF;

  IF v_queue_status <> 'active' THEN
    RETURN jsonb_build_object('success', false, 'error', 'already_settled');
  END IF;

  -- No bids placed — mark unsold
  IF v_winner_id IS NULL THEN
    UPDATE public.auction_queue
    SET    status = 'unsold', ended_at = now()
    WHERE  id = p_auction_queue_id;

    INSERT INTO public.auction_events (auction_id, event_type, payload_json)
    VALUES (
      v_auction_id, 'item_unsold',
      jsonb_build_object(
        'auction_queue_id', p_auction_queue_id,
        'player_id',        v_player_id
      )
    )
    RETURNING seq INTO v_event_seq;

    RETURN jsonb_build_object(
      'success', true, 'outcome', 'unsold', 'event_seq', v_event_seq
    );
  END IF;

  -- Mark sold
  UPDATE public.auction_queue
  SET    status = 'sold', ended_at = now()
  WHERE  id = p_auction_queue_id;

  -- Deduct winning bid from the winner's budget (fails if budget is now insufficient)
  UPDATE public.league_users
  SET    remaining_budget = remaining_budget - v_winning_bid,
         version          = version + 1
  WHERE  league_id = v_league_id AND user_id = v_winner_id
    AND  remaining_budget >= v_winning_bid;

  IF NOT FOUND THEN
    RAISE EXCEPTION
      'Budget check failed during settlement for user % in league %',
      v_winner_id, v_league_id;
  END IF;

  -- Record player ownership (ignore if somehow already owned — shouldn't happen)
  INSERT INTO public.player_ownerships (league_id, user_id, player_id, purchase_price)
  VALUES (v_league_id, v_winner_id, v_player_id, v_winning_bid)
  ON CONFLICT (league_id, player_id) DO NOTHING;

  -- Record the financial transaction
  INSERT INTO public.transactions
    (league_id, user_id, amount, type, reference_id, description)
  VALUES (
    v_league_id, v_winner_id, -v_winning_bid, 'bid_win',
    p_auction_queue_id::text,
    'Auction purchase'
  );

  -- Append settlement event to the log
  INSERT INTO public.auction_events (auction_id, event_type, payload_json)
  VALUES (
    v_auction_id, 'item_sold',
    jsonb_build_object(
      'auction_queue_id', p_auction_queue_id,
      'player_id',        v_player_id,
      'winner_id',        v_winner_id,
      'price',            v_winning_bid
    )
  )
  RETURNING seq INTO v_event_seq;

  RETURN jsonb_build_object(
    'success',   true,
    'outcome',   'sold',
    'winner_id', v_winner_id,
    'price',     v_winning_bid,
    'event_seq', v_event_seq
  );
END;
$$;

COMMENT ON FUNCTION public.settle_auction_item IS
  'Closes an active auction item atomically. Triggered by the Go auction clock, never directly by clients.';

-- ─────────────────────────────────────────────────────────────
-- 18. Function: get_missed_events
--
--     Called by a reconnecting client to catch up on events it missed.
--     The client passes the seq of the last event it processed; this
--     returns every subsequent event in order so the UI can reconcile.
-- ─────────────────────────────────────────────────────────────
CREATE OR REPLACE FUNCTION public.get_missed_events(
  p_auction_id uuid,
  p_after_seq  bigint
)
RETURNS SETOF public.auction_events
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public
AS $$
  SELECT *
  FROM   public.auction_events
  WHERE  auction_id = p_auction_id
    AND  seq > p_after_seq
  ORDER  BY seq ASC;
$$;

COMMENT ON FUNCTION public.get_missed_events IS
  'Returns all auction_events for the given auction with seq > p_after_seq. Used for state reconciliation on WebSocket reconnect.';
