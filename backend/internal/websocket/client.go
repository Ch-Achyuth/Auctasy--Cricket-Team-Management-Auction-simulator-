package websocket

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// writeWait is the deadline for a single WebSocket write to complete.
	writeWait = 10 * time.Second

	// pongWait is how long the server waits for a pong before declaring the client dead.
	pongWait = 60 * time.Second

	// pingPeriod is how often the server sends a ping. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize caps inbound frames. Clients are observers — inbound messages
	// are discarded, but the limit prevents a misbehaving client from allocating
	// unbounded memory.
	maxMessageSize = 4 * 1024 // 4 KiB

	// sendBufSize is the depth of each client's outbound message queue.
	// If it fills up the client is evicted by the hub as too slow.
	sendBufSize = 256
)

// Client represents a single browser's WebSocket connection to one league room.
//
// A Client is a read-only observer of the auction:
//   - readPump drains and discards all inbound frames (clients never push auction state).
//   - writePump dequeues messages from the hub and writes them to the browser.
//
// This strict separation ensures the WebSocket layer never writes to the database.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan []byte // buffered outbound message queue; closed by the hub on eviction
	leagueID string
	userID   string
}

// ServeWS upgrades an HTTP connection to WebSocket and registers the new Client
// with the hub. The caller is responsible for JWT authentication and must supply
// the verified userID and leagueID — this function trusts them unconditionally.
//
// allowedOrigin is the permitted value of the Origin header (e.g. "http://localhost:8000").
// Pass an empty string in development to allow all origins.
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, userID, leagueID, allowedOrigin string) {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			if allowedOrigin == "" {
				return true
			}
			return r.Header.Get("Origin") == allowedOrigin
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[ws] upgrade failed user=%s league=%s: %v", userID, leagueID, err)
		return
	}

	c := &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, sendBufSize),
		leagueID: leagueID,
		userID:   userID,
	}

	// Register with the hub before launching pumps so the client is in the room
	// before any broadcast can target it.
	hub.register <- c

	// Each pump runs in its own goroutine.
	// writePump is the only goroutine that calls conn.WriteMessage — gorilla/websocket
	// requires that no more than one goroutine writes to a connection at a time.
	go c.writePump()
	go c.readPump()
}

// readPump drains inbound WebSocket frames and honours the ping/pong heartbeat.
// Inbound messages are discarded: this is a broadcast-only connection.
//
// When the client disconnects (or the read deadline expires), readPump sends
// the Client to hub.unregister and closes the underlying connection.
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		// Reset the deadline on every pong — proves the client is still alive.
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		// Read and discard — clients observe the auction, they do not push state.
		if _, _, err := c.conn.ReadMessage(); err != nil {
			if websocket.IsUnexpectedCloseError(err,
				websocket.CloseGoingAway,
				websocket.CloseAbnormalClosure,
			) {
				log.Printf("[ws] unexpected close user=%s league=%s: %v",
					c.userID, c.leagueID, err)
			}
			return
		}
	}
}

// writePump dequeues messages from the hub's send channel and writes them to
// the WebSocket. It is the sole writer to the connection (gorilla/websocket
// requirement). It also drives the ping heartbeat via a ticker.
//
// writePump exits when:
//   - the hub closes the send channel (eviction or shutdown), or
//   - a write to the connection fails (network error / client gone).
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the send channel — send a WebSocket close frame.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// Each AuctionEvent is its own WebSocket text frame so the client can
			// JSON.parse(e.data) safely. Coalescing multiple JSON objects into one
			// frame with '\n' separators breaks the client-side parser.
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}

		case <-ticker.C:
			// Send a ping to detect dead connections before the OS TCP timeout.
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
