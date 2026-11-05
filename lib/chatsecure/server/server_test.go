package server

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"torbox/chatsecure/storage"
)

func generateTestPGPKey(name, email string) (string, string, error) {
	entity, err := openpgp.NewEntity(name, "test comment", email, nil)
	if err != nil {
		return "", "", err
	}

	for _, id := range entity.Identities {
		err := id.SelfSignature.SignUserId(id.UserId.Id, entity.PrimaryKey, entity.PrivateKey, nil)
		if err != nil {
			return "", "", err
		}
	}

	var pubBuf bytes.Buffer
	w, err := armor.Encode(&pubBuf, openpgp.PublicKeyType, nil)
	if err != nil {
		return "", "", err
	}
	err = entity.Serialize(w)
	if err != nil {
		return "", "", err
	}
	_ = w.Close()

	fp := fmt.Sprintf("%X", entity.PrimaryKey.Fingerprint)
	return pubBuf.String(), fp, nil
}

func generateSyntheticEd25519Key(name, email string) (string, string, error) {
	var pubBody bytes.Buffer
	pubBody.WriteByte(4)
	_ = binary.Write(&pubBody, binary.BigEndian, uint32(1700000000))
	pubBody.WriteByte(22)
	oid := []byte{0x2b, 0x06, 0x01, 0x04, 0x01, 0xda, 0x47, 0x0f, 0x01}
	pubBody.WriteByte(byte(len(oid)))
	pubBody.Write(oid)
	_ = binary.Write(&pubBody, binary.BigEndian, uint16(263))
	pubBody.WriteByte(0x40)
	pubBody.Write(make([]byte, 32))

	h := sha1.New()
	h.Write([]byte{0x99, byte(pubBody.Len() >> 8), byte(pubBody.Len())})
	h.Write(pubBody.Bytes())
	fp := fmt.Sprintf("%X", h.Sum(nil))

	var packetData bytes.Buffer
	packetData.WriteByte(0xC0 | 6)
	if pubBody.Len() < 192 {
		packetData.WriteByte(byte(pubBody.Len()))
	} else {
		packetData.WriteByte(byte((pubBody.Len()-192)>>8 + 192))
		packetData.WriteByte(byte((pubBody.Len() - 192) & 0xFF))
	}
	packetData.Write(pubBody.Bytes())

	uidStr := fmt.Sprintf("%s <%s>", name, email)
	packetData.WriteByte(0xC0 | 13)
	packetData.WriteByte(byte(len(uidStr)))
	packetData.WriteString(uidStr)

	var armoredBuf bytes.Buffer
	w, err := armor.Encode(&armoredBuf, openpgp.PublicKeyType, nil)
	if err != nil {
		return "", "", err
	}
	_, _ = w.Write(packetData.Bytes())
	_ = w.Close()

	return armoredBuf.String(), fp, nil
}

func setupTestApp(t *testing.T, instName string) (http.Handler, *storage.Storage, *Hub, string) {
	store, err := storage.InitDB(instName, false)
	if err != nil {
		t.Fatalf("failed to init db: %v", err)
	}

	hub := NewHub(store, false)
	go hub.Run()

	secretKey := "test-secret-key-123456"
	cfg := ServerConfig{
		BaseDir:     "..",
		OnionDomain: "testchat.onion",
		SecretKey:   secretKey,
		Dev:         true,
		Debug:       false,
	}

	handler := NewServer(cfg, store, hub)
	return handler, store, hub, secretKey
}

func TestPGPKeyParsing(t *testing.T) {
	pubArmor, expectedFP, err := generateTestPGPKey("Alice", "alice@test.onion")
	if err != nil {
		t.Fatalf("generateTestPGPKey failed: %v", err)
	}

	info, err := ParsePGPPublicKey(pubArmor)
	if err != nil {
		t.Fatalf("ParsePGPPublicKey failed: %v", err)
	}

	if info.Fingerprint != expectedFP {
		t.Errorf("expected fingerprint %s, got %s", expectedFP, info.Fingerprint)
	}
	if info.Name != "Alice" {
		t.Errorf("expected name 'Alice', got '%s'", info.Name)
	}

	edArmor, edFP, err := generateSyntheticEd25519Key("OperadorEd", "operador@torbox.onion")
	if err != nil {
		t.Fatalf("generateSyntheticEd25519Key failed: %v", err)
	}

	edInfo, err := ParsePGPPublicKey(edArmor)
	if err != nil {
		t.Fatalf("ParsePGPPublicKey for Ed25519 failed: %v", err)
	}
	if edInfo.Fingerprint != edFP {
		t.Errorf("expected fingerprint %s, got %s", edFP, edInfo.Fingerprint)
	}
	if edInfo.Name != "OperadorEd" {
		t.Errorf("expected name 'OperadorEd', got '%s'", edInfo.Name)
	}
}

func TestAPIEndToEnd(t *testing.T) {
	handler, store, _, _ := setupTestApp(t, "test_api_suite")
	defer store.Close()

	aliceKey, _, err := generateTestPGPKey("Alice", "alice@onion.net")
	if err != nil {
		t.Fatalf("failed Alice key: %v", err)
	}
	bobKey, _, err := generateTestPGPKey("Bob", "bob@onion.net")
	if err != nil {
		t.Fatalf("failed Bob key: %v", err)
	}

	loginPayload := map[string]string{
		"name":   "Alice",
		"pubkey": aliceKey,
	}
	body, _ := json.Marshal(loginPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed: code %d, body: %s", w.Code, w.Body.String())
	}

	var aliceAuth struct {
		ID    int64  `json:"id"`
		Token string `json:"token"`
	}
	_ = json.NewDecoder(w.Body).Decode(&aliceAuth)

	reqRe := httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(body))
	reqRe.Header.Set("Content-Type", "application/json")
	wRe := httptest.NewRecorder()
	handler.ServeHTTP(wRe, reqRe)

	var aliceAuth2 struct {
		ID    int64  `json:"id"`
		Token string `json:"token"`
	}
	_ = json.NewDecoder(wRe.Body).Decode(&aliceAuth2)
	if aliceAuth2.ID != aliceAuth.ID || aliceAuth2.Token != aliceAuth.Token {
		t.Errorf("re-login returned mismatch credentials")
	}

	bobPayload, _ := json.Marshal(map[string]string{
		"name":   "Bob",
		"pubkey": bobKey,
	})
	reqBob := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(bobPayload))
	reqBob.Header.Set("Content-Type", "application/json")
	wBob := httptest.NewRecorder()
	handler.ServeHTTP(wBob, reqBob)

	if wBob.Code != http.StatusOK {
		t.Fatalf("login Bob failed: %d", wBob.Code)
	}
	var bobAuth struct {
		ID    int64  `json:"id"`
		Token string `json:"token"`
	}
	_ = json.NewDecoder(wBob.Body).Decode(&bobAuth)

	aliceCloneKey, _, _ := generateTestPGPKey("Alice", "otheralice@onion.net")
	clonePayload, _ := json.Marshal(map[string]string{
		"name":   "Alice",
		"pubkey": aliceCloneKey,
	})
	reqClone := httptest.NewRequest(http.MethodPost, "/api/login", bytes.NewReader(clonePayload))
	reqClone.Header.Set("Content-Type", "application/json")
	wClone := httptest.NewRecorder()
	handler.ServeHTTP(wClone, reqClone)

	if wClone.Code != http.StatusForbidden {
		t.Errorf("expected 403 Forbidden for duplicate name, got %d", wClone.Code)
	}

	reqUsers := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	reqUsers.Header.Set("Authorization", "Token "+bobAuth.Token)
	wUsers := httptest.NewRecorder()
	handler.ServeHTTP(wUsers, reqUsers)

	if wUsers.Code != http.StatusOK {
		t.Fatalf("get users failed: %d", wUsers.Code)
	}
	var users []storage.UserWithLastMsg
	_ = json.NewDecoder(wUsers.Body).Decode(&users)
	if len(users) != 1 || users[0].ID != aliceAuth.ID {
		t.Errorf("expected Bob to see 1 user (Alice), got: %+v", users)
	}

	sendPayload, _ := json.Marshal(map[string]any{
		"client_msg_id": "msg-uuid-001",
		"recipient":     bobAuth.ID,
		"is_group":      false,
		"msg":           "-----BEGIN PGP MESSAGE-----\nciphertext1\n-----END PGP MESSAGE-----",
	})
	reqSend := httptest.NewRequest(http.MethodPost, "/api/messages/send", bytes.NewReader(sendPayload))
	reqSend.Header.Set("Authorization", "token "+aliceAuth.Token)
	reqSend.Header.Set("Content-Type", "application/json")
	wSend := httptest.NewRecorder()
	handler.ServeHTTP(wSend, reqSend)

	if wSend.Code != http.StatusOK {
		t.Fatalf("send message failed: %d, body: %s", wSend.Code, wSend.Body.String())
	}
	var ack struct {
		ID          int64  `json:"id"`
		ClientMsgID string `json:"client_msg_id"`
		TS          string `json:"ts"`
	}
	_ = json.NewDecoder(wSend.Body).Decode(&ack)
	if ack.ID == 0 || ack.ClientMsgID != "msg-uuid-001" {
		t.Errorf("invalid send ack: %+v", ack)
	}

	groupPayload, _ := json.Marshal(map[string]any{
		"client_msg_id": "group-msg-002",
		"recipient":     "default",
		"is_group":      true,
		"msg":           "-----BEGIN PGP MESSAGE-----\ngrouptext1\n-----END PGP MESSAGE-----",
	})
	reqGroup := httptest.NewRequest(http.MethodPost, "/api/messages/send", bytes.NewReader(groupPayload))
	reqGroup.Header.Set("Authorization", "Token "+bobAuth.Token)
	reqGroup.Header.Set("Content-Type", "application/json")
	wGroup := httptest.NewRecorder()
	handler.ServeHTTP(wGroup, reqGroup)

	if wGroup.Code != http.StatusOK {
		t.Fatalf("send group message failed: %d", wGroup.Code)
	}
	var groupAck struct {
		ID          int64  `json:"id"`
		ClientMsgID string `json:"client_msg_id"`
	}
	_ = json.NewDecoder(wGroup.Body).Decode(&groupAck)
	if groupAck.ID <= ack.ID {
		t.Errorf("expected monotonic group message ID > %d, got %d", ack.ID, groupAck.ID)
	}

	reqSync := httptest.NewRequest(http.MethodGet, "/api/messages/sync?since_id=0&group_id=default", nil)
	reqSync.Header.Set("Authorization", "token "+bobAuth.Token)
	wSync := httptest.NewRecorder()
	handler.ServeHTTP(wSync, reqSync)

	if wSync.Code != http.StatusOK {
		t.Fatalf("sync messages failed: %d", wSync.Code)
	}
	var syncRes struct {
		Messages []struct {
			ID        int64  `json:"id"`
			Sender    int64  `json:"sender"`
			Recipient any    `json:"recipient"`
			Msg       string `json:"msg"`
			TS        string `json:"ts"`
		} `json:"messages"`
		LastID int64 `json:"last_id"`
	}
	_ = json.NewDecoder(wSync.Body).Decode(&syncRes)

	if len(syncRes.Messages) != 2 {
		t.Fatalf("expected 2 synced messages, got %d", len(syncRes.Messages))
	}
	if syncRes.LastID != groupAck.ID {
		t.Errorf("expected last_id=%d, got %d", groupAck.ID, syncRes.LastID)
	}

	reqHistory := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/user_msg/%d", aliceAuth.ID), nil)
	reqHistory.Header.Set("Authorization", "Token "+bobAuth.Token)
	wHistory := httptest.NewRecorder()
	handler.ServeHTTP(wHistory, reqHistory)
	if wHistory.Code != http.StatusOK {
		t.Errorf("expected 200 from /user_msg, got %d", wHistory.Code)
	}

	reqGHistory := httptest.NewRequest(http.MethodGet, "/group_msg", nil)
	reqGHistory.Header.Set("Authorization", "Token "+aliceAuth.Token)
	wGHistory := httptest.NewRecorder()
	handler.ServeHTTP(wGHistory, reqGHistory)
	if wGHistory.Code != http.StatusOK {
		t.Errorf("expected 200 from /group_msg, got %d", wGHistory.Code)
	}

	reqRoot := httptest.NewRequest(http.MethodGet, "/", nil)
	wRoot := httptest.NewRecorder()
	handler.ServeHTTP(wRoot, reqRoot)
	if wRoot.Code != http.StatusOK {
		t.Fatalf("root index.html request failed: %d", wRoot.Code)
	}
	if !strings.Contains(wRoot.Body.String(), "<html") {
		t.Errorf("expected HTML in root response")
	}
}

func TestHubMessageBroadcasting(t *testing.T) {
	_, store, hub, _ := setupTestApp(t, "test_hub_suite")
	defer store.Close()

	u1, err := store.CreateUser("WSUser1", "-----BEGIN PGP PUBLIC KEY BLOCK-----\nwstest1", "FP_WS_USER_1", "token_ws_1")
	if err != nil {
		t.Fatalf("CreateUser 1 failed: %v", err)
	}
	u2, err := store.CreateUser("WSUser2", "-----BEGIN PGP PUBLIC KEY BLOCK-----\nwstest2", "FP_WS_USER_2", "token_ws_2")
	if err != nil {
		t.Fatalf("CreateUser 2 failed: %v", err)
	}

	client1 := &Client{
		hub:        hub,
		send:       make(chan []byte, 10),
		user:       u1,
		isSocketIO: true,
	}
	hub.RegisterClient(client1)
	time.Sleep(20 * time.Millisecond)

	client2 := &Client{
		hub:        hub,
		send:       make(chan []byte, 10),
		user:       u2,
		isSocketIO: true,
	}
	hub.RegisterClient(client2)
	time.Sleep(20 * time.Millisecond)

	// Client 1 should receive user_connected for Client 2
	select {
	case msg := <-client1.send:
		if !strings.Contains(string(msg), "user_connected") {
			t.Errorf("expected user_connected broadcast, got: %s", string(msg))
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for user_connected broadcast on client 1")
	}

	// Drain any connection events on client 2
	select {
	case <-client2.send:
	case <-time.After(100 * time.Millisecond):
	}

	// Now User 1 sends a direct 1-to-1 message to User 2
	dm := &storage.Message{
		ID:        100,
		Sender:    u1.ID,
		Recipient: u2.ID,
		Msg:       "secret_dm_message",
		TS:        time.Now(),
		IsGroup:   false,
	}
	hub.BroadcastChatMessage(dm, u2.FP)

	// Client 2 MUST receive the message packet
	select {
	case msg := <-client2.send:
		str := string(msg)
		if !strings.Contains(str, "secret_dm_message") {
			t.Errorf("expected secret_dm_message on client 2, got: %s", str)
		}
		if !strings.Contains(str, `"message"`) {
			t.Errorf("expected 'message' Socket.IO event name, got: %s", str)
		}
		if !strings.Contains(str, fmt.Sprintf(`"recipient":%d`, u2.ID)) {
			t.Errorf("expected recipient=%d in payload, got: %s", u2.ID, str)
		}
		if !strings.Contains(str, fmt.Sprintf(`"sender":%d`, u1.ID)) {
			t.Errorf("expected sender=%d in payload, got: %s", u1.ID, str)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timeout waiting for DM on client 2")
	}

	hub.UnregisterClient(client1)
	hub.UnregisterClient(client2)
}

func TestTwoClientsGroupChatAndRefresh(t *testing.T) {
	handler, store, hub, _ := setupTestApp(t, "test_chat_refresh_suite")
	defer store.Close()

	u1, err := store.CreateUser("Operador1", "-----BEGIN PGP PUBLIC KEY BLOCK-----\nop1", "FP_OP_1", "token_op_1")
	if err != nil {
		t.Fatalf("CreateUser 1 failed: %v", err)
	}
	u2, err := store.CreateUser("Operador2", "-----BEGIN PGP PUBLIC KEY BLOCK-----\nop2", "FP_OP_2", "token_op_2")
	if err != nil {
		t.Fatalf("CreateUser 2 failed: %v", err)
	}

	client1 := &Client{
		hub:        hub,
		send:       make(chan []byte, 10),
		user:       u1,
		isSocketIO: true,
	}
	hub.RegisterClient(client1)

	client2 := &Client{
		hub:        hub,
		send:       make(chan []byte, 10),
		user:       u2,
		isSocketIO: true,
	}
	hub.RegisterClient(client2)

	// 1. Client 1 MUST receive user_connected event for Client 2
	select {
	case msg := <-client1.send:
		str := string(msg)
		if !strings.Contains(str, "user_connected") {
			t.Fatalf("expected user_connected for client 2, got: %s", str)
		}
		if !strings.Contains(str, `"name":"Operador2"`) {
			t.Fatalf("expected Operador2 in payload, got: %s", str)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for user_connected")
	}

	// 2. Client 2 sends a group message
	client2.processMessage("default", "Mensaje de prueba en grupo", true, "c_123")

	// 3. Client 1 MUST receive the message
	select {
	case msg := <-client1.send:
		str := string(msg)
		if !strings.Contains(str, `"message"`) {
			t.Fatalf("expected 'message' Socket.IO event, got: %s", str)
		}
		if !strings.Contains(str, "Mensaje de prueba en grupo") {
			t.Fatalf("expected message body, got: %s", str)
		}
		if !strings.Contains(str, `"recipient":"default"`) {
			t.Fatalf("expected recipient 'default', got: %s", str)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for group message on client 1")
	}

	// Client 2 also receives group echo (ChatContext ignores own messages via if (value.sender !== userId))
	select {
	case <-client2.send:
	case <-time.After(100 * time.Millisecond):
	}

	// 4. Client 1 simulates a browser refresh: calls GET /users and GET /group_msg
	reqUsers := httptest.NewRequest(http.MethodGet, "/users", nil)
	reqUsers.Header.Set("Authorization", "Token token_op_1")
	wUsers := httptest.NewRecorder()
	handler.ServeHTTP(wUsers, reqUsers)
	if wUsers.Code != http.StatusOK {
		t.Fatalf("expected 200 from /users, got %d", wUsers.Code)
	}

	var usersList []map[string]any
	if err := json.Unmarshal(wUsers.Body.Bytes(), &usersList); err != nil {
		t.Fatalf("failed to decode /users response: %v", err)
	}
	if len(usersList) != 1 {
		t.Fatalf("expected 1 contact (Operador2), got %d", len(usersList))
	}
	if usersList[0]["name"] != "Operador2" {
		t.Errorf("expected contact to be Operador2, got %v", usersList[0]["name"])
	}

	// 5. Check GET /group_msg on refresh
	reqGMsg := httptest.NewRequest(http.MethodGet, "/group_msg", nil)
	reqGMsg.Header.Set("Authorization", "Token token_op_1")
	wGMsg := httptest.NewRecorder()
	handler.ServeHTTP(wGMsg, reqGMsg)
	if wGMsg.Code != http.StatusOK {
		t.Fatalf("expected 200 from /group_msg, got %d", wGMsg.Code)
	}

	var groupMsgs []map[string]any
	if err := json.Unmarshal(wGMsg.Body.Bytes(), &groupMsgs); err != nil {
		t.Fatalf("failed to decode /group_msg response: %v", err)
	}
	if len(groupMsgs) != 1 {
		t.Fatalf("expected 1 group message on refresh, got %d", len(groupMsgs))
	}
	if groupMsgs[0]["msg"] != "Mensaje de prueba en grupo" {
		t.Errorf("expected 'Mensaje de prueba en grupo', got %v", groupMsgs[0]["msg"])
	}

	// 6. When Client 1 disconnects, Client 2 receives user_disconnected
	hub.UnregisterClient(client1)
	select {
	case msg := <-client2.send:
		str := string(msg)
		if !strings.Contains(str, "user_disconnected") {
			t.Fatalf("expected user_disconnected, got: %s", str)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("timed out waiting for user_disconnected")
	}

	// 7. But Client 2 can STILL see Client 1 in /users (marked inactive)
	reqUsersAfterDisconnect := httptest.NewRequest(http.MethodGet, "/users", nil)
	reqUsersAfterDisconnect.Header.Set("Authorization", "Token token_op_2")
	wUsersAfter := httptest.NewRecorder()
	handler.ServeHTTP(wUsersAfter, reqUsersAfterDisconnect)
	var usersListAfter []map[string]any
	_ = json.Unmarshal(wUsersAfter.Body.Bytes(), &usersListAfter)
	if len(usersListAfter) != 1 {
		t.Fatalf("expected Client 1 to still be in contacts, got %d", len(usersListAfter))
	}
	if usersListAfter[0]["active"] != false {
		t.Errorf("expected Client 1 to have active=false, got %v", usersListAfter[0]["active"])
	}

	hub.UnregisterClient(client2)
}

