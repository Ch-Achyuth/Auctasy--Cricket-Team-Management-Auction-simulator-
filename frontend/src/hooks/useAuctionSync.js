import { useState, useEffect, useRef } from 'react'

const MIN_RECONNECT_MS = 1000
const MAX_RECONNECT_MS = 10000
const API_BASE = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'
const WS_BASE = API_BASE.replace(/^http/, 'ws')

/**
 * useAuctionSync — live WebSocket connection to an auction room.
 *
 * @param {string|null}   leagueId  League room to subscribe to.
 * @param {object|null}   session   Supabase session from useAuth().
 * @param {Function}      onEvent   Called with each AuctionEvent in seq order.
 * @returns {{ isSyncing: boolean }}
 */
export function useAuctionSync(leagueId, session, onEvent) {
  const [isSyncing, setIsSyncing] = useState(false)

  // Highest seq number seen — persists across reconnects without triggering renders.
  const lastSeqRef = useRef(0)

  // Always-current callback ref: avoids recreating the WebSocket on every render
  // when the caller passes an inline function for onEvent.
  const onEventRef = useRef(onEvent)
  useEffect(() => { onEventRef.current = onEvent })

  // Ref mirror of isSyncing so the synchronous onmessage closure reads the
  // live value rather than the stale one captured at socket creation time.
  const isSyncingRef = useRef(false)

  function setSyncing(val) {
    isSyncingRef.current = val
    setIsSyncing(val)
  }

  const accessToken = session?.access_token ?? null

  useEffect(() => {
    if (!leagueId || !accessToken) return

    let active = true
    let ws = null
    let reconnectTimer = null
    let reconnectDelay = MIN_RECONNECT_MS
    // Events arriving over the WebSocket during the catch-up fetch are buffered
    // here and replayed after WAL events are applied, so nothing is lost or
    // applied out of order.
    const buffer = []

    function applyEvent(event) {
      if (typeof event?.seq !== 'number') return
      // seq deduplication: catch-up and live streams can overlap on reconnect.
      if (event.seq <= lastSeqRef.current) return
      lastSeqRef.current = event.seq
      onEventRef.current?.(event)
    }

    async function catchUp() {
      const url =
        `${API_BASE}/api/v1/auction/missed-events` +
        `?league_id=${encodeURIComponent(leagueId)}` +
        `&after_seq=${lastSeqRef.current}`
      const res = await fetch(url, {
        headers: { Authorization: `Bearer ${accessToken}` },
      })
      if (!res.ok) throw new Error(`missed-events returned ${res.status}`)
      const events = await res.json()
      if (Array.isArray(events)) {
        for (const event of events) applyEvent(event)
      }
    }

    function openSocket() {
      if (!active) return

      const wsUrl =
        `${WS_BASE}/ws` +
        `?token=${encodeURIComponent(accessToken)}` +
        `&league_id=${encodeURIComponent(leagueId)}`

      ws = new WebSocket(wsUrl)

      ws.onopen = async () => {
        if (!active) { ws.close(); return }

        reconnectDelay = MIN_RECONNECT_MS
        setSyncing(true)
        buffer.length = 0

        try {
          await catchUp()
          // Replay messages that arrived while the fetch was in-flight.
          for (const event of buffer) applyEvent(event)
        } catch (err) {
          console.error('[useAuctionSync] catch-up failed:', err)
          // Still drain the buffer — a partial state is better than a frozen one.
          for (const event of buffer) applyEvent(event)
        } finally {
          buffer.length = 0
          if (active) setSyncing(false)
        }
      }

      ws.onmessage = (e) => {
        let event
        try { event = JSON.parse(e.data) } catch { return }

        if (isSyncingRef.current) {
          buffer.push(event)
        } else {
          applyEvent(event)
        }
      }

      ws.onclose = () => {
        if (!active) return
        // Double the delay before scheduling so the next onclose fires at the
        // new (longer) interval, not the one used to reach the failed attempt.
        const delay = reconnectDelay
        reconnectDelay = Math.min(reconnectDelay * 2, MAX_RECONNECT_MS)
        reconnectTimer = setTimeout(() => {
          if (active) openSocket()
        }, delay)
      }

      ws.onerror = (err) => {
        console.error('[useAuctionSync] WebSocket error:', err)
        // onclose fires after onerror; reconnect is scheduled there.
      }
    }

    openSocket()

    return () => {
      active = false
      clearTimeout(reconnectTimer)
      if (ws) {
        // Null out onclose BEFORE calling close() so the reconnect timer is not
        // scheduled during deliberate teardown (unmount or dependency change).
        ws.onclose = null
        ws.close()
      }
      if (isSyncingRef.current) setSyncing(false)
    }
  }, [leagueId, accessToken])

  return { isSyncing }
}
