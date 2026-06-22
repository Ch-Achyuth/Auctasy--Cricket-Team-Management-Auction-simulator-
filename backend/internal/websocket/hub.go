// Package websocket implements the real-time broadcast layer for live auctions.
//
// Responsibilities (strictly enforced):
//   - Upgrade HTTP connections to WebSocket and manage their lifecycle.
//   - Subscribe to Redis Pub/Sub channels per league and fan out events to
//     all locally connected clients.
//   - Expose Publish() so REST/RPC handlers can push AuctionEvents after a
//     successful database write — the hub itself never touches the database.
package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const redisChannelPrefix = "auction_room:"

// AuctionEvent mirrors the auction_events WAL table structure.
// The hub treats it as opaque bytes — it only marshals/unmarshals the envelope.
// No business logic or database writes happen in this package.
type AuctionEvent struct {
	ID          string          `json:"id"`
	AuctionID   string          `json:"auction_id"`
	EventType   string          `json:"event_type"`
	PayloadJSON json.RawMessage `json:"payload_json"`
	Seq         int64           `json:"seq"`
	CreatedAt   time.Time       `json:"created_at"`
}

// envelope is an internal message routed from a Redis subscriber goroutine to
// the Run() loop's broadcast case, binding a raw JSON payload to its league room.
type envelope struct {
	leagueID string
	data     []byte
}

// leagueSub bundles the Redis PubSub handle with a stop signal so the
// subscriber goroutine can be preempted whether it is waiting on Redis or
// blocked on the broadcast channel.
type leagueSub struct {
	pubsub *redis.PubSub
	done   chan struct{}
}

// Hub maintains all active WebSocket clients partitioned by league_id.
// It bridges Redis Pub/Sub messages to every connected browser in a league room.
//
// Concurrency model
//
//	┌─ Run() goroutine ─────────────────────────────────────────────────┐
//	│  Owns all writes to `rooms`. Serialises register / unregister /   │
//	│  broadcast via channels so no two operations race on the map.     │
//	│  Also starts/stops Redis subscriber goroutines synchronously.     │
//	└───────────────────────────────────────────────────────────────────┘
//	  mu (RWMutex) — allows any goroutine to call RoomSize() safely
//	                 without a channel round-trip.
//	  subsMu (Mutex) — guards `subs` independently; acquired only for
//	                   subscribe/unsubscribe, never held across I/O.
type Hub struct {
	// rooms[leagueID] = set of connected clients; guarded by mu.
	rooms map[string]map[*Client]struct{}
	mu    sync.RWMutex

	// All mutations to rooms flow through these channels into Run().
	register   chan *Client
	unregister chan *Client
	// broadcast delivers a JSON payload to every client in one league room.
	broadcast chan envelope

	redis  *redis.Client
	subs   map[string]*leagueSub // leagueID → active subscription; guarded by subsMu
	subsMu sync.Mutex
}

// NewHub creates a Hub backed by the provided Redis client.
// Call hub.Run(ctx) in a dedicated goroutine before registering any clients.
func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]struct{}),
		register:   make(chan *Client, 64),
		unregister: make(chan *Client, 64),
		broadcast:  make(chan envelope, 512),
		redis:      rdb,
		subs:       make(map[string]*leagueSub),
	}
}

// Run is the Hub's single-threaded event loop. Start it once in a goroutine.
//
// It is the only writer to the rooms map. The RWMutex lets external callers
// (metrics, HTTP handlers) read room state without routing through channels.
func (h *Hub) Run(ctx context.Context) {
	for {
		select {

		case client := <-h.register:
			h.mu.Lock()
			isNew := false
			if _, ok := h.rooms[client.leagueID]; !ok {
				h.rooms[client.leagueID] = make(map[*Client]struct{})
				isNew = true
			}
			h.rooms[client.leagueID][client] = struct{}{}
			h.mu.Unlock()

			// Start the Redis subscription only when the first client joins.
			// Done after the lock release to avoid holding mu across network I/O.
			if isNew {
				h.subscribeLeague(ctx, client.leagueID)
			}
			log.Printf("[hub] register user=%s league=%s room_size=%d",
				client.userID, client.leagueID, h.RoomSize(client.leagueID))

		case client := <-h.unregister:
			h.mu.Lock()
			isEmpty := false
			if room, ok := h.rooms[client.leagueID]; ok {
				if _, inRoom := room[client]; inRoom {
					delete(room, client)
					close(client.send)
					if len(room) == 0 {
						delete(h.rooms, client.leagueID)
						isEmpty = true
					}
				}
			}
			h.mu.Unlock()

			// Tear down the Redis subscription only when the room is empty.
			// Called synchronously in Run() — no concurrent subscribe can race here.
			if isEmpty {
				h.unsubscribeLeague(client.leagueID)
			}
			log.Printf("[hub] unregister user=%s league=%s", client.userID, client.leagueID)

		case msg := <-h.broadcast:
			// Snapshot the target set under a read lock so we don't hold it
			// while doing channel sends, which may block.
			h.mu.RLock()
			room := h.rooms[msg.leagueID]
			targets := make([]*Client, 0, len(room))
			for c := range room {
				targets = append(targets, c)
			}
			h.mu.RUnlock()

			for _, c := range targets {
				select {
				case c.send <- msg.data:
				default:
					// Client send buffer full — it is too slow or disconnected.
					// Evict in-place; avoids a re-entry through the unregister channel.
					h.evict(c)
				}
			}

		case <-ctx.Done():
			log.Println("[hub] shutdown: context cancelled")
			return
		}
	}
}

// evict removes a stalled client from its room and closes its send channel.
// Must only be called from within the Run() goroutine (i.e. the broadcast case).
func (h *Hub) evict(c *Client) {
	h.mu.Lock()
	room, ok := h.rooms[c.leagueID]
	if !ok {
		h.mu.Unlock()
		return
	}
	if _, inRoom := room[c]; !inRoom {
		h.mu.Unlock()
		return
	}
	delete(room, c)
	close(c.send) // writePump sees closed channel and sends CloseMessage
	isEmpty := len(room) == 0
	if isEmpty {
		delete(h.rooms, c.leagueID)
	}
	h.mu.Unlock()

	if isEmpty {
		// Safe to call synchronously — evict is already in Run(), and
		// unsubscribeLeague only touches subsMu (not mu or channels).
		h.unsubscribeLeague(c.leagueID)
	}
	log.Printf("[hub] evicted slow/dead client user=%s league=%s", c.userID, c.leagueID)
}

// subscribeLeague starts a goroutine that forwards Redis Pub/Sub messages for
// the given league into the hub's broadcast channel.
// Must only be called from within the Run() goroutine.
func (h *Hub) subscribeLeague(ctx context.Context, leagueID string) {
	channel := redisChannelPrefix + leagueID
	pubsub := h.redis.Subscribe(ctx, channel)
	sub := &leagueSub{
		pubsub: pubsub,
		done:   make(chan struct{}),
	}

	h.subsMu.Lock()
	h.subs[leagueID] = sub
	h.subsMu.Unlock()

	go func() {
		redisCh := pubsub.Channel()
		for {
			select {
			case msg, ok := <-redisCh:
				if !ok {
					// PubSub closed by unsubscribeLeague.
					return
				}
				// Forward to the hub's broadcast loop.
				// Select on done so the goroutine is preemptable even when
				// the broadcast channel is at capacity.
				select {
				case h.broadcast <- envelope{leagueID: leagueID, data: []byte(msg.Payload)}:
				case <-sub.done:
					return
				case <-ctx.Done():
					return
				}
			case <-sub.done:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	log.Printf("[hub] subscribed redis channel=%s", channel)
}

// unsubscribeLeague stops the Redis subscriber goroutine for the given league.
// Must only be called from within the Run() goroutine (unregister case or evict).
func (h *Hub) unsubscribeLeague(leagueID string) {
	h.subsMu.Lock()
	sub, ok := h.subs[leagueID]
	if ok {
		delete(h.subs, leagueID)
	}
	h.subsMu.Unlock()

	if !ok {
		return
	}
	// Signal the subscriber goroutine to exit before closing the PubSub,
	// in case it is blocked on a broadcast channel send.
	close(sub.done)
	sub.pubsub.Close()
	log.Printf("[hub] unsubscribed redis channel=%s%s", redisChannelPrefix, leagueID)
}

// Publish marshals an AuctionEvent and pushes it to the Redis Pub/Sub channel
// for the given league. All hub instances subscribed to that channel (including
// this one) will fan out the event to their local WebSocket clients.
//
// This is the sole entry point for REST/RPC handlers to notify connected clients.
// The hub itself never reads from or writes to the database.
func (h *Hub) Publish(ctx context.Context, leagueID string, event AuctionEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return h.redis.Publish(ctx, redisChannelPrefix+leagueID, string(data)).Err()
}

// RoomSize returns the number of clients currently in the given league room.
// Safe to call from any goroutine.
func (h *Hub) RoomSize(leagueID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.rooms[leagueID])
}
