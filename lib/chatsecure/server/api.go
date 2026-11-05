package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"torbox/chatsecure/storage"
)

type APIHandler struct {
	storage   *storage.Storage
	hub       *Hub
	secretKey string
	debug     bool
}

func NewAPIHandler(s *storage.Storage, hub *Hub, secretKey string, debug bool) *APIHandler {
	return &APIHandler{
		storage:   s,
		hub:       hub,
		secretKey: secretKey,
		debug:     debug,
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// HandleLogin handles POST /login and POST /api/login
func (api *APIHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name   string `json:"name"`
		Pubkey string `json:"pubkey"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Pubkey) == "" {
		writeError(w, http.StatusBadRequest, "pubkey is required")
		return
	}

	keyInfo, err := ParsePGPPublicKey(req.Pubkey)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid PGP public key: "+err.Error())
		return
	}

	name := keyInfo.Name
	if name == "" {
		name = strings.TrimSpace(req.Name)
	}
	if name == "" {
		writeError(w, http.StatusBadRequest, "User name could not be determined")
		return
	}

	fp := keyInfo.Fingerprint

	// 1. Check if user with this fingerprint already exists
	existingUser, err := api.storage.GetUserByFP(fp)
	if err == nil && existingUser != nil {
		_ = api.storage.UpdateUserStatus(existingUser.ID, true, nil)
		writeJSON(w, http.StatusOK, map[string]any{
			"id":    existingUser.ID,
			"token": existingUser.Token,
		})
		return
	}

	// 2. Check if active user with this name already exists
	sameNameUser, err := api.storage.GetUserByName(name)
	if err == nil && sameNameUser != nil && sameNameUser.Active {
		writeError(w, http.StatusForbidden, "username in use")
		return
	}

	// 3. Generate token = sha1(fp + SECRET_KEY)
	token := GenerateToken(fp, api.secretKey)

	// 4. Create new user in RAM DB
	newUser, err := api.storage.CreateUser(name, req.Pubkey, fp, token)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to register user: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":    newUser.ID,
		"token": token,
	})
}

// HandleUsers handles GET /users and GET /api/users
func (api *APIHandler) HandleUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := AuthenticateRequest(r, api.storage)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	users, err := api.storage.GetActiveUsers(user.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, users)
}

// HandleGroups handles GET /groups and GET /api/groups
func (api *APIHandler) HandleGroups(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	_, err := AuthenticateRequest(r, api.storage)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	groups := []map[string]any{
		{
			"id":   1,
			"name": "default",
		},
	}
	writeJSON(w, http.StatusOK, groups)
}

// HandleSendMessage handles POST /api/messages/send
func (api *APIHandler) HandleSendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := AuthenticateRequest(r, api.storage)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	var req struct {
		ClientMsgID string `json:"client_msg_id"`
		Recipient   any    `json:"recipient"`
		IsGroup     bool   `json:"is_group"`
		Msg         string `json:"msg"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	if strings.TrimSpace(req.Msg) == "" {
		writeError(w, http.StatusBadRequest, "Message body cannot be empty")
		return
	}

	if req.IsGroup {
		savedMsg, err := api.storage.SaveGroupMessage(user.ID, req.Msg)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to save message: "+err.Error())
			return
		}
		savedMsg.ClientMsgID = req.ClientMsgID
		api.hub.BroadcastChatMessage(savedMsg, "")

		writeJSON(w, http.StatusOK, map[string]any{
			"id":            savedMsg.ID,
			"client_msg_id": req.ClientMsgID,
			"ts":            savedMsg.TS.Format(time.RFC3339),
		})
		return
	}

	// 1-to-1 message
	var recipID int64
	switch v := req.Recipient.(type) {
	case float64:
		recipID = int64(v)
	case string:
		recipID, _ = strconv.ParseInt(v, 10, 64)
	}

	if recipID == 0 {
		writeError(w, http.StatusBadRequest, "Invalid recipient ID")
		return
	}

	recipUser, err := api.storage.GetUserByID(recipID)
	if err != nil || recipUser == nil {
		writeError(w, http.StatusNotFound, "Recipient not found")
		return
	}

	savedMsg, err := api.storage.SaveUserMessage(user.ID, recipID, req.Msg)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to save message: "+err.Error())
		return
	}
	savedMsg.ClientMsgID = req.ClientMsgID
	api.hub.BroadcastChatMessage(savedMsg, recipUser.FP)

	writeJSON(w, http.StatusOK, map[string]any{
		"id":            savedMsg.ID,
		"client_msg_id": req.ClientMsgID,
		"ts":            savedMsg.TS.Format(time.RFC3339),
	})
}

// HandleSyncMessages handles GET /api/messages/sync
func (api *APIHandler) HandleSyncMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := AuthenticateRequest(r, api.storage)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	sinceIDStr := r.URL.Query().Get("since_id")
	sinceID, _ := strconv.ParseInt(sinceIDStr, 10, 64)
	groupID := r.URL.Query().Get("group_id")
	if groupID == "" {
		groupID = "default"
	}

	messages, lastID, err := api.storage.SyncMessages(user.ID, sinceID, groupID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Sync error: "+err.Error())
		return
	}

	type syncItem struct {
		ID        int64  `json:"id"`
		Sender    int64  `json:"sender"`
		Recipient any    `json:"recipient"`
		Msg       string `json:"msg"`
		TS        string `json:"ts"`
	}

	items := make([]syncItem, 0, len(messages))
	for _, m := range messages {
		items = append(items, syncItem{
			ID:        m.ID,
			Sender:    m.Sender,
			Recipient: m.Recipient,
			Msg:       m.Msg,
			TS:        m.TS.Format("2006-01-02 15:04:05"),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"messages": items,
		"last_id":  lastID,
	})
}

// HandleUserMessages handles GET /user_msg/{sender_id} and GET /api/messages/user/{sender_id}
func (api *APIHandler) HandleUserMessages(w http.ResponseWriter, r *http.Request, senderIDStr string) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := AuthenticateRequest(r, api.storage)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	senderID, err := strconv.ParseInt(senderIDStr, 10, 64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Invalid sender ID")
		return
	}

	messages, err := api.storage.GetUserMessages(user.ID, senderID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	type msgItem struct {
		ID        int64  `json:"id"`
		Sender    int64  `json:"sender"`
		Recipient any    `json:"recipient"`
		Msg       string `json:"msg"`
		TS        string `json:"ts"`
	}

	items := make([]msgItem, 0, len(messages))
	for _, m := range messages {
		items = append(items, msgItem{
			ID:        m.ID,
			Sender:    m.Sender,
			Recipient: m.Recipient,
			Msg:       m.Msg,
			TS:        m.TS.Format("2006-01-02 15:04:05"),
		})
	}

	writeJSON(w, http.StatusOK, items)
}

// HandleGroupMessages handles GET /group_msg and GET /api/messages/group/default
func (api *APIHandler) HandleGroupMessages(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	user, err := AuthenticateRequest(r, api.storage)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}

	messages, err := api.storage.GetGroupMessages(user.TsJoin, 100)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Database error: "+err.Error())
		return
	}

	type msgItem struct {
		ID        int64  `json:"id"`
		Sender    int64  `json:"sender"`
		Recipient any    `json:"recipient"`
		Msg       string `json:"msg"`
		TS        string `json:"ts"`
	}

	items := make([]msgItem, 0, len(messages))
	for _, m := range messages {
		items = append(items, msgItem{
			ID:        m.ID,
			Sender:    m.Sender,
			Recipient: m.Recipient,
			Msg:       m.Msg,
			TS:        m.TS.Format("2006-01-02 15:04:05"),
		})
	}

	writeJSON(w, http.StatusOK, items)
}
