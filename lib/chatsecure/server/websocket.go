package server

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"torbox/chatsecure/storage"
)

const (
	writeWait      = 15 * time.Second
	pongWait       = 120 * time.Second
	pingPeriod     = 60 * time.Second
	maxMessageSize = 512 * 1024 // 512 KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 64,
	WriteBufferSize: 1024 * 64,
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for Onion services
	},
}

// Client represents a connected WebSocket peer
type Client struct {
	hub          *Hub
	conn         *websocket.Conn
	send         chan []byte
	user         *storage.User
	isSocketIO   bool
	debug        bool
	sid          string
	skipAnnounce bool // set when the polling session already broadcast user_connected
}

type PollingSession struct {
	id        string
	user      *storage.User
	client    *Client // linked websocket client if upgraded
	sendChan  chan string
	lastSeen  time.Time
	connected bool
	announced bool // true once user_connected has been broadcast for this session
}

// Hub maintains active connections and handles message routing
type Hub struct {
	storage         *storage.Storage
	clients         map[*Client]bool
	userRooms       map[string]map[*Client]bool // room name -> clients
	pollingSessions map[string]*PollingSession
	broadcast       chan []byte
	mu              sync.RWMutex
	pollingMu       sync.Mutex
	debug           bool
}

func NewHub(s *storage.Storage, debug bool) *Hub {
	h := &Hub{
		storage:         s,
		clients:         make(map[*Client]bool),
		userRooms:       make(map[string]map[*Client]bool),
		pollingSessions: make(map[string]*PollingSession),
		broadcast:       make(chan []byte, 256),
		debug:           debug,
	}

	// Clean up stale polling sessions periodically
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			h.cleanupPollingSessions()
		}
	}()

	return h
}

func (h *Hub) Run() {
	// Hub event loop
	for msg := range h.broadcast {
		h.mu.RLock()
		for client := range h.clients {
			select {
			case client.send <- msg:
			default:
			}
		}
		h.mu.RUnlock()
	}
}

func (h *Hub) cleanupPollingSessions() {
	h.pollingMu.Lock()
	defer h.pollingMu.Unlock()

	now := time.Now()
	for id, sess := range h.pollingSessions {
		if now.Sub(sess.lastSeen) > 90*time.Second {
			delete(h.pollingSessions, id)
		}
	}
}

func (h *Hub) RegisterClient(client *Client) {
	if client == nil {
		return
	}

	h.mu.Lock()
	h.clients[client] = true
	var newlyJoinedUser *storage.User

	if client.user != nil {
		alreadyInGroup := false
		if h.userRooms["default"] != nil && h.userRooms["default"][client] {
			alreadyInGroup = true
		}

		h.joinRoomLocked("default", client)
		h.joinRoomLocked(client.user.FP, client)
		h.joinRoomLocked(strconv.FormatInt(client.user.ID, 10), client)

		sid := fmt.Sprintf("%p", client)
		_ = h.storage.UpdateUserStatus(client.user.ID, true, &sid)

		if !alreadyInGroup && !client.skipAnnounce {
			newlyJoinedUser = client.user
		}
	}
	h.mu.Unlock()

	if newlyJoinedUser != nil {
		if h.debug {
			log.Printf("[hub] User %s (id=%d, fp=%s) connected. Broadcasting user_connected.", newlyJoinedUser.Name, newlyJoinedUser.ID, newlyJoinedUser.FP)
		}
		h.BroadcastUserStatusExcept(client, newlyJoinedUser, true)
	}
}

func (h *Hub) UnregisterClient(client *Client) {
	if client == nil {
		return
	}

	h.mu.Lock()
	var disconnectedUser *storage.User

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		if client.user != nil {
			h.leaveRoomLocked("default", client)
			h.leaveRoomLocked(client.user.FP, client)
			h.leaveRoomLocked(strconv.FormatInt(client.user.ID, 10), client)

			hasOtherConns := false
			if personalClients, exists := h.userRooms[client.user.FP]; exists && len(personalClients) > 0 {
				hasOtherConns = true
			}

			if !hasOtherConns {
				_ = h.storage.UpdateUserStatus(client.user.ID, false, nil)
				disconnectedUser = client.user
			}
		}
	}
	h.mu.Unlock()

	if disconnectedUser != nil {
		if h.debug {
			log.Printf("[hub] User %s (id=%d) disconnected. Broadcasting user_disconnected.", disconnectedUser.Name, disconnectedUser.ID)
		}
		h.BroadcastUserStatusExcept(client, disconnectedUser, false)
	}
}

func (h *Hub) joinRoomLocked(room string, client *Client) {
	if h.userRooms[room] == nil {
		h.userRooms[room] = make(map[*Client]bool)
	}
	h.userRooms[room][client] = true
}

func (h *Hub) leaveRoomLocked(room string, client *Client) {
	if clients, exists := h.userRooms[room]; exists {
		delete(clients, client)
		if len(clients) == 0 {
			delete(h.userRooms, room)
		}
	}
}

// BroadcastUserStatus notifies all connected clients of user connect/disconnect
func (h *Hub) BroadcastUserStatus(u *storage.User, connected bool) {
	h.BroadcastUserStatusExcept(nil, u, connected)
}

// BroadcastUserStatusExcept notifies connected clients of user status, excluding the specified client.
// IMPORTANT: Must NOT be called while holding h.mu or h.pollingMu to avoid deadlocks.
func (h *Hub) BroadcastUserStatusExcept(exclude *Client, u *storage.User, connected bool) {
	var eventName string
	var payload any

	if connected {
		eventName = "user_connected"
		payload = map[string]any{
			"id":          u.ID,
			"name":        u.Name,
			"fp":          u.FP,
			"pubkey":      u.Pubkey,
			"last_update": u.LastUpdate.Format("2006-01-02 15:04:05"),
			"active":      true,
			"msg":         nil,
		}
	} else {
		eventName = "user_disconnected"
		payload = map[string]any{
			"id":   u.ID,
			"name": u.Name,
		}
	}

	rawJSON, _ := json.Marshal(map[string]any{
		"event": eventName,
		"data":  payload,
	})

	sioJSON, _ := json.Marshal([]any{eventName, payload})
	sioPacket := []byte(fmt.Sprintf("42%s", string(sioJSON)))

	// Snapshot WebSocket clients under read lock, then send without holding the lock.
	h.mu.RLock()
	targets := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		if exclude != nil && client == exclude {
			continue
		}
		if connected && client.user != nil && client.user.ID == u.ID {
			continue
		}
		targets = append(targets, client)
	}
	h.mu.RUnlock()

	for _, client := range targets {
		var toSend []byte
		if client.isSocketIO {
			toSend = sioPacket
		} else {
			toSend = rawJSON
		}
		select {
		case client.send <- toSend:
		default:
		}
	}

	// Snapshot polling sessions under their own lock, then send without holding it.
	h.pollingMu.Lock()
	type pollTarget struct {
		sendChan chan string
	}
	var pollTargets []pollTarget
	for _, sess := range h.pollingSessions {
		if connected && sess.user != nil && sess.user.ID == u.ID {
			continue
		}
		pollTargets = append(pollTargets, pollTarget{sendChan: sess.sendChan})
	}
	h.pollingMu.Unlock()

	for _, pt := range pollTargets {
		select {
		case pt.sendChan <- string(sioPacket):
		default:
		}
	}
}

// BroadcastChatMessage delivers a message to the target room or user.
// IMPORTANT: Must NOT be called while holding h.mu or h.pollingMu to avoid deadlocks.
func (h *Hub) BroadcastChatMessage(msg *storage.Message, recipientFP string) {
	msgData := map[string]any{
		"id":            msg.ID,
		"sender":        msg.Sender,
		"recipient":     msg.Recipient,
		"msg":           msg.Msg,
		"ts":            msg.TS.Format("2006-01-02 15:04:05"),
		"client_msg_id": msg.ClientMsgID,
		"is_group":      msg.IsGroup,
	}

	rawJSON, _ := json.Marshal(map[string]any{
		"event": "new_message",
		"data":  msgData,
	})

	// Socket.IO event: 'message' as expected by ChatContext.jsx
	sioJSON, _ := json.Marshal([]any{"message", msgData})
	sioPacket := []byte(fmt.Sprintf("42%s", string(sioJSON)))

	var recipID int64

	// Snapshot WebSocket targets under read lock, then release before sending.
	h.mu.RLock()
	targetClients := make(map[*Client]bool)
	if msg.IsGroup {
		for cl := range h.userRooms["default"] {
			targetClients[cl] = true
		}
		for cl := range h.clients {
			if cl.user != nil {
				targetClients[cl] = true
			}
		}
	} else {
		switch v := msg.Recipient.(type) {
		case int64:
			recipID = v
		case int:
			recipID = int64(v)
		case float64:
			recipID = int64(v)
		case string:
			recipID, _ = strconv.ParseInt(v, 10, 64)
		}

		recipStr := fmt.Sprintf("%v", msg.Recipient)
		senderStr := strconv.FormatInt(msg.Sender, 10)

		// Add recipient clients by numeric ID
		for cl := range h.userRooms[recipStr] {
			targetClients[cl] = true
		}
		// Add recipient clients by fingerprint
		if recipientFP != "" {
			for cl := range h.userRooms[recipientFP] {
				targetClients[cl] = true
			}
		}
		// Also add sender's other connections so all user's devices/tabs stay in sync
		for cl := range h.userRooms[senderStr] {
			targetClients[cl] = true
		}
	}
	h.mu.RUnlock()

	if h.debug {
		log.Printf("[hub] Broadcasting message (id=%d, isGroup=%v, recip=%v) to %d WebSocket clients",
			msg.ID, msg.IsGroup, msg.Recipient, len(targetClients))
	}

	for client := range targetClients {
		var toSend []byte
		if client.isSocketIO {
			toSend = sioPacket
		} else {
			toSend = rawJSON
		}

		select {
		case client.send <- toSend:
		default:
		}
	}

	// Snapshot polling sessions under their own lock, then send without holding it.
	h.pollingMu.Lock()
	type pollTarget struct {
		sendChan chan string
	}
	var pollTargets []pollTarget
	for _, sess := range h.pollingSessions {
		if sess.user != nil {
			if msg.IsGroup || sess.user.ID == recipID || (recipientFP != "" && sess.user.FP == recipientFP) {
				pollTargets = append(pollTargets, pollTarget{sendChan: sess.sendChan})
			}
		}
	}
	h.pollingMu.Unlock()

	for _, pt := range pollTargets {
		select {
		case pt.sendChan <- string(sioPacket):
		default:
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		c.hub.UnregisterClient(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				if c.debug {
					log.Printf("[ws] Unexpected close: %v", err)
				}
			}
			break
		}

		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		c.handleIncoming(message)
	}
}

func (c *Client) handleIncoming(raw []byte) {
	str := string(raw)

	// Handle Socket.IO / Engine.IO protocol
	if c.isSocketIO {
		// Engine.IO pong: client's response to our server-side "2" ping.
		// The read deadline was already reset in readPump before calling here.
		if str == "3" {
			return
		}

		// Engine.IO ping from client (v3-style compatibility).
		if str == "2" {
			c.send <- []byte("3")
			return
		}

		// Engine.IO upgrade probe: "2probe" -> reply with "3probe"
		if str == "2probe" {
			c.send <- []byte("3probe")
			return
		}

		// Engine.IO upgrade acknowledgment: "5"
		if str == "5" {
			return
		}

		// Connect packet with auth: e.g. 40{"token":"..."}, 40,{"token":"..."}, 40/{"token":"..."}
		if strings.HasPrefix(str, "40") {
			var token string
			if start := strings.Index(str, "{"); start != -1 {
				if end := strings.LastIndex(str, "}"); end > start {
					var authData struct {
						Token string `json:"token"`
					}
					if err := json.Unmarshal([]byte(str[start:end+1]), &authData); err == nil && authData.Token != "" {
						token = authData.Token
					}
				}
			}

			if token != "" {
				if u, err := c.hub.storage.GetUserByToken(token); err == nil && u != nil {
					c.user = u
					c.hub.RegisterClient(c)
				}
			}
			c.send <- []byte("40{\"sid\":\"tcs-session\"}")
			return
		}

		// Event packet: 42["event", {...}]
		if strings.HasPrefix(str, "42") {
			eventPayload := strings.TrimPrefix(str, "42")
			// Remove any namespace prefix like /namespace,
			if start := strings.Index(eventPayload, "["); start != -1 {
				eventPayload = eventPayload[start:]
			}

			var parts []json.RawMessage
			if err := json.Unmarshal([]byte(eventPayload), &parts); err == nil && len(parts) >= 2 {
				var eventName string
				_ = json.Unmarshal(parts[0], &eventName)

				if eventName == "msg" {
					var data struct {
						Sender    int64  `json:"sender"`
						Recipient any    `json:"recipient"`
						Msg       string `json:"msg"`
						IsGroup   bool   `json:"is_group"`
					}
					if err := json.Unmarshal(parts[1], &data); err == nil {
						// If user was not yet attached to client, try to resolve by sender ID
						if c.user == nil && data.Sender > 0 {
							if u, err := c.hub.storage.GetUserByID(data.Sender); err == nil && u != nil {
								c.user = u
								c.hub.RegisterClient(c)
							}
						}

						if c.user != nil {
							c.processMessage(data.Recipient, data.Msg, data.IsGroup, "")
						}
					}
				}
			}
			return
		}
	}

	// Native WebSocket JSON handling
	var parsed struct {
		Action      string `json:"action"`
		Event       string `json:"event"`
		Token       string `json:"token"`
		Recipient   any    `json:"recipient"`
		Msg         string `json:"msg"`
		IsGroup     bool   `json:"is_group"`
		ClientMsgID string `json:"client_msg_id"`
	}

	if err := json.Unmarshal(raw, &parsed); err == nil {
		if parsed.Token != "" && c.user == nil {
			u, err := c.hub.storage.GetUserByToken(parsed.Token)
			if err == nil && u != nil {
				c.user = u
				c.hub.RegisterClient(c)
			}
		}

		if (parsed.Action == "send" || parsed.Event == "msg") && c.user != nil {
			c.processMessage(parsed.Recipient, parsed.Msg, parsed.IsGroup, parsed.ClientMsgID)
		}
	}
}

func (c *Client) processMessage(recipient any, msgText string, isGroup bool, clientMsgID string) {
	if isGroup {
		savedMsg, err := c.hub.storage.SaveGroupMessage(c.user.ID, msgText)
		if err == nil {
			savedMsg.ClientMsgID = clientMsgID
			c.hub.BroadcastChatMessage(savedMsg, "")
		}
	} else {
		var recipID int64
		var recipFP string
		switch v := recipient.(type) {
		case float64:
			recipID = int64(v)
		case int64:
			recipID = v
		case int:
			recipID = int64(v)
		case string:
			recipID, _ = strconv.ParseInt(v, 10, 64)
		}

		recipUser, err := c.hub.storage.GetUserByID(recipID)
		if err == nil && recipUser != nil {
			recipFP = recipUser.FP
		}

		savedMsg, err := c.hub.storage.SaveUserMessage(c.user.ID, recipID, msgText)
		if err == nil {
			savedMsg.ClientMsgID = clientMsgID
			c.hub.BroadcastChatMessage(savedMsg, recipFP)
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if c.isSocketIO {
				// Engine.IO v4 application-level ping (text "2").
				// The Socket.IO client expects THIS, not a WebSocket ping frame.
				// Client must respond with text "3" within pongWait, or we disconnect.
				if err := c.conn.WriteMessage(websocket.TextMessage, []byte("2")); err != nil {
					return
				}
			} else {
				// Native WebSocket ping frame for /ws clients.
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		}
	}
}

// ServeWs handles standard WebSocket requests on /ws
func (h *Hub) ServeWs(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		if h.debug {
			log.Printf("[ws] Upgrade error: %v", err)
		}
		return
	}

	client := &Client{
		hub:        h,
		conn:       conn,
		send:       make(chan []byte, 256),
		isSocketIO: false,
		debug:      h.debug,
	}

	token := ExtractToken(r)
	if token != "" {
		u, err := h.storage.GetUserByToken(token)
		if err == nil && u != nil {
			client.user = u
		}
	}

	h.RegisterClient(client)

	go client.writePump()
	go client.readPump()
}

func newSessionID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// ServeSocketIO handles Socket.IO v4 protocol (long-polling + websocket upgrade)
func (h *Hub) ServeSocketIO(w http.ResponseWriter, r *http.Request) {
	transport := r.URL.Query().Get("transport")
	sid := r.URL.Query().Get("sid")

	// Try extracting user from cookie, header, or query param
	var initialUser *storage.User
	token := ExtractToken(r)
	if token != "" {
		if u, err := h.storage.GetUserByToken(token); err == nil && u != nil {
			initialUser = u
		}
	}

	// 1. WebSocket upgrade or direct connection
	if transport == "websocket" || r.Header.Get("Upgrade") == "websocket" {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			if h.debug {
				log.Printf("[socketio-ws] Upgrade error: %v", err)
			}
			return
		}

		client := &Client{
			hub:        h,
			conn:       conn,
			send:       make(chan []byte, 256),
			isSocketIO: true,
			debug:      h.debug,
			sid:        sid,
		}

		if initialUser != nil {
			client.user = initialUser
		}

		// Always check the polling session (even when initialUser is set from cookie)
		// so we can propagate sess.announced → client.skipAnnounce and avoid a
		// duplicate user_connected broadcast when the polling POST already announced it.
		if sid != "" {
			h.pollingMu.Lock()
			if sess, ok := h.pollingSessions[sid]; ok {
				sess.client = client
				if sess.user != nil && client.user == nil {
					client.user = sess.user
				}
				if sess.announced {
					client.skipAnnounce = true
				}
			}
			h.pollingMu.Unlock()
		}

		h.RegisterClient(client)

		go client.writePump()
		go client.readPump()
		return
	}

	// 2. HTTP Long-Polling Transport
	w.Header().Set("Content-Type", "text/plain; charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// 2A. Initial Handshake (GET without sid)
	if sid == "" && r.Method == http.MethodGet {
		newSID := newSessionID()
		sess := &PollingSession{
			id:        newSID,
			sendChan:  make(chan string, 32),
			lastSeen:  time.Now(),
			connected: false,
			user:      initialUser,
		}

		// NOTE: Do NOT broadcast user_connected here. The handshake only creates the
		// Engine.IO session; user identity is confirmed later via the Socket.IO connect
		// packet (POST 40{token}). Broadcasting here always causes a duplicate because
		// the WS upgrade fires RegisterClient milliseconds later.
		if initialUser != nil {
			sess.connected = true
			_ = h.storage.UpdateUserStatus(initialUser.ID, true, &newSID)
		}

		h.pollingMu.Lock()
		h.pollingSessions[newSID] = sess
		h.pollingMu.Unlock()

		// pingInterval / pingTimeout must match the server's pingPeriod (60s) and pongWait (120s).
		handshake := fmt.Sprintf("0{\"sid\":\"%s\",\"upgrades\":[\"websocket\"],\"pingInterval\":60000,\"pingTimeout\":120000,\"maxPayload\":1000000}", newSID)
		_, _ = w.Write([]byte(handshake))
		return
	}

	// Look up session
	// NOTE: Do NOT call BroadcastUserStatus while holding pollingMu — deadlock.
	// Snapshot what we need, release the lock, then broadcast.
	var userToAnnounce *storage.User
	h.pollingMu.Lock()
	sess, exists := h.pollingSessions[sid]
	if exists {
		sess.lastSeen = time.Now()
		if sess.user == nil && initialUser != nil {
			sess.user = initialUser
			sess.connected = true
			_ = h.storage.UpdateUserStatus(initialUser.ID, true, &sid)
			// Mark announced so the POST 40{token} handler doesn't also broadcast.
			sess.announced = true
			userToAnnounce = initialUser
		}
	}
	h.pollingMu.Unlock()

	if userToAnnounce != nil {
		h.BroadcastUserStatus(userToAnnounce, true)
	}

	if !exists {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("{\"code\":1,\"message\":\"Session ID unknown\"}"))
		return
	}

	// 2B. Client POST (sends data like 40 connect or 42 msg)
	if r.Method == http.MethodPost {
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		bodyStr := string(bodyBytes)

		// Connect packet: 40{"token":"..."}
		if strings.HasPrefix(bodyStr, "40") {
			var postToken string
			if start := strings.Index(bodyStr, "{"); start != -1 {
				if end := strings.LastIndex(bodyStr, "}"); end > start {
					var authData struct {
						Token string `json:"token"`
					}
					if err := json.Unmarshal([]byte(bodyStr[start:end+1]), &authData); err == nil && authData.Token != "" {
						postToken = authData.Token
					}
				}
			}

			if postToken != "" {
				if u, err := h.storage.GetUserByToken(postToken); err == nil && u != nil {
					sess.user = u
					sess.connected = true
					_ = h.storage.UpdateUserStatus(u.ID, true, &sess.id)
					if !sess.announced {
						// First time this session's user is confirmed — announce them.
						sess.announced = true
						if sess.client != nil {
							// WS client already connected; tell it to skip its own broadcast.
							sess.client.skipAnnounce = true
							sess.client.user = u
							h.RegisterClient(sess.client)
						} else {
							// Polling-only path; WS upgrade will see sess.announced=true.
							h.BroadcastUserStatus(u, true)
						}
					} else if sess.client != nil && sess.client.user == nil {
						// Already announced (e.g. via lookup broadcast), but WS client
						// connected without a user set yet — fix that up silently.
						sess.client.user = u
					}
				}
			}
			// Queue connect ACK
			select {
			case sess.sendChan <- fmt.Sprintf("40{\"sid\":\"%s\"}", sess.id):
			default:
			}
		} else if strings.HasPrefix(bodyStr, "42") {
			eventPayload := strings.TrimPrefix(bodyStr, "42")
			if start := strings.Index(eventPayload, "["); start != -1 {
				eventPayload = eventPayload[start:]
			}
			var parts []json.RawMessage
			if err := json.Unmarshal([]byte(eventPayload), &parts); err == nil && len(parts) >= 2 {
				var eventName string
				_ = json.Unmarshal(parts[0], &eventName)
				if eventName == "msg" {
					var data struct {
						Sender    int64  `json:"sender"`
						Recipient any    `json:"recipient"`
						Msg       string `json:"msg"`
						IsGroup   bool   `json:"is_group"`
					}
					if err := json.Unmarshal(parts[1], &data); err == nil {
						if sess.user == nil && data.Sender > 0 {
							if u, err := h.storage.GetUserByID(data.Sender); err == nil && u != nil {
								sess.user = u
							}
						}
						if sess.user != nil {
							if data.IsGroup {
								savedMsg, err := h.storage.SaveGroupMessage(sess.user.ID, data.Msg)
								if err == nil {
									h.BroadcastChatMessage(savedMsg, "")
								}
							} else {
								var recipID int64
								var recipFP string
								switch v := data.Recipient.(type) {
								case float64:
									recipID = int64(v)
								case int64:
									recipID = v
								case int:
									recipID = int64(v)
								case string:
									recipID, _ = strconv.ParseInt(v, 10, 64)
								}
								recipUser, err := h.storage.GetUserByID(recipID)
								if err == nil && recipUser != nil {
									recipFP = recipUser.FP
								}
								savedMsg, err := h.storage.SaveUserMessage(sess.user.ID, recipID, data.Msg)
								if err == nil {
									h.BroadcastChatMessage(savedMsg, recipFP)
								}
							}
						}
					}
				}
			}
		}

		_, _ = w.Write([]byte("ok"))
		return
	}

	// 2C. Client GET (Long-polling for incoming packets)
	select {
	case msg := <-sess.sendChan:
		_, _ = w.Write([]byte(msg))
		return
	default:
	}

	select {
	case msg := <-sess.sendChan:
		_, _ = w.Write([]byte(msg))
	case <-time.After(25 * time.Second):
		_, _ = w.Write([]byte("6"))
	case <-r.Context().Done():
		return
	}
}
