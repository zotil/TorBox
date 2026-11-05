package storage

import (
	"testing"
	"time"
)

func TestStorageInitAndCleanup(t *testing.T) {
	store, err := InitDB("test_instance", true)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer store.Close()

	// Verify default group exists
	var groupName string
	err = store.DB().QueryRow("SELECT name FROM groups WHERE id = 1").Scan(&groupName)
	if err != nil {
		t.Fatalf("default group query failed: %v", err)
	}
	if groupName != "default" {
		t.Errorf("expected group 'default', got '%s'", groupName)
	}
}

func TestStorageUsersAndMessages(t *testing.T) {
	store, err := InitDB("test_users", true)
	if err != nil {
		t.Fatalf("InitDB failed: %v", err)
	}
	defer store.Close()

	// Create User 1
	u1, err := store.CreateUser("Operador1", "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest1", "FP1111111111111111111111111111111111111111", "token1")
	if err != nil {
		t.Fatalf("CreateUser 1 failed: %v", err)
	}

	// Create User 2
	u2, err := store.CreateUser("Operador2", "-----BEGIN PGP PUBLIC KEY BLOCK-----\ntest2", "FP2222222222222222222222222222222222222222", "token2")
	if err != nil {
		t.Fatalf("CreateUser 2 failed: %v", err)
	}

	// Test GetUserByToken
	found, err := store.GetUserByToken("token1")
	if err != nil || found.ID != u1.ID {
		t.Fatalf("GetUserByToken failed: %v", err)
	}

	// Test GetUserByFP
	foundFP, err := store.GetUserByFP("FP2222222222222222222222222222222222222222")
	if err != nil || foundFP.ID != u2.ID {
		t.Fatalf("GetUserByFP failed: %v", err)
	}

	// Test Direct Message
	m1, err := store.SaveUserMessage(u1.ID, u2.ID, "encrypted_dm_1")
	if err != nil {
		t.Fatalf("SaveUserMessage failed: %v", err)
	}
	if m1.ID <= 0 {
		t.Errorf("expected positive message ID, got %d", m1.ID)
	}

	// Test GetUserMessages
	msgs, err := store.GetUserMessages(u1.ID, u2.ID)
	if err != nil || len(msgs) != 1 {
		t.Fatalf("GetUserMessages failed: len=%d, err=%v", len(msgs), err)
	}
	if msgs[0].Msg != "encrypted_dm_1" {
		t.Errorf("unexpected message content: %s", msgs[0].Msg)
	}

	// Test Group Message
	gm1, err := store.SaveGroupMessage(u1.ID, "encrypted_group_msg")
	if err != nil {
		t.Fatalf("SaveGroupMessage failed: %v", err)
	}

	// Test GetGroupMessages
	gmsgs, err := store.GetGroupMessages(time.Now().Add(-1*time.Hour), 10)
	if err != nil || len(gmsgs) != 1 {
		t.Fatalf("GetGroupMessages failed: len=%d, err=%v", len(gmsgs), err)
	}

	// Test SyncMessages (Cursor Sync Fallback)
	syncMsgs, lastID, err := store.SyncMessages(u2.ID, 0, "default")
	if err != nil {
		t.Fatalf("SyncMessages failed: %v", err)
	}
	if len(syncMsgs) != 2 { // 1 DM + 1 Group
		t.Fatalf("expected 2 synced messages, got %d", len(syncMsgs))
	}
	if lastID != gm1.ID {
		t.Errorf("expected lastID=%d, got %d", gm1.ID, lastID)
	}

	// Sync with cursor past first message
	syncSince, _, err := store.SyncMessages(u2.ID, m1.ID, "default")
	if err != nil {
		t.Fatalf("SyncMessages since m1 failed: %v", err)
	}
	if len(syncSince) != 1 {
		t.Errorf("expected 1 message in syncSince, got %d", len(syncSince))
	}
	// Test GetActiveUsers
	activeUsers, err := store.GetActiveUsers(u1.ID)
	if err != nil {
		t.Fatalf("GetActiveUsers failed: %v", err)
	}
	if len(activeUsers) != 1 || activeUsers[0].ID != u2.ID {
		t.Fatalf("expected 1 active user (u2), got %d, list=%+v", len(activeUsers), activeUsers)
	}
}
