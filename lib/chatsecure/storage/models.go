package storage

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type User struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Pubkey     string    `json:"pubkey"`
	FP         string    `json:"fp"`
	Token      string    `json:"token"`
	SID        *string   `json:"sid,omitempty"`
	TsJoin     time.Time `json:"ts_join"`
	LastUpdate time.Time `json:"last_update"`
	Active     bool      `json:"active"`
}

type UserWithLastMsg struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	FP         string  `json:"fp"`
	Pubkey     string  `json:"pubkey"`
	LastUpdate string  `json:"last_update"`
	Active     bool    `json:"active"`
	Msg        *string `json:"msg"`
}

type Group struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	LastUpdate time.Time `json:"last_update"`
}

type Message struct {
	ID           int64     `json:"id"`
	Sender       int64     `json:"sender"`
	Recipient    any       `json:"recipient"` // string "default" for group, or int64 for direct message
	Msg          string    `json:"msg"`
	TS           time.Time `json:"ts"`
	IsGroup      bool      `json:"is_group,omitempty"`
	ClientMsgID  string    `json:"client_msg_id,omitempty"`
}

// GetUserByToken looks up a user by authentication token.
func (s *Storage) GetUserByToken(token string) (*User, error) {
	row := s.db.QueryRow(`
		SELECT id, name, pubkey, fp, token, sid, ts_join, last_update, active
		FROM users
		WHERE token = ?`, token)

	u := &User{}
	var sid sql.NullString
	var tsJoin, lastUpdate string
	err := row.Scan(&u.ID, &u.Name, &u.Pubkey, &u.FP, &u.Token, &sid, &tsJoin, &lastUpdate, &u.Active)
	if err != nil {
		return nil, err
	}
	if sid.Valid {
		u.SID = &sid.String
	}
	u.TsJoin, _ = parseTime(tsJoin)
	u.LastUpdate, _ = parseTime(lastUpdate)
	return u, nil
}

// GetUserByFP looks up a user by PGP fingerprint.
func (s *Storage) GetUserByFP(fp string) (*User, error) {
	row := s.db.QueryRow(`
		SELECT id, name, pubkey, fp, token, sid, ts_join, last_update, active
		FROM users
		WHERE fp = ?`, fp)

	u := &User{}
	var sid sql.NullString
	var tsJoin, lastUpdate string
	err := row.Scan(&u.ID, &u.Name, &u.Pubkey, &u.FP, &u.Token, &sid, &tsJoin, &lastUpdate, &u.Active)
	if err != nil {
		return nil, err
	}
	if sid.Valid {
		u.SID = &sid.String
	}
	u.TsJoin, _ = parseTime(tsJoin)
	u.LastUpdate, _ = parseTime(lastUpdate)
	return u, nil
}

// GetUserByName looks up an active user by username.
func (s *Storage) GetUserByName(name string) (*User, error) {
	row := s.db.QueryRow(`
		SELECT id, name, pubkey, fp, token, sid, ts_join, last_update, active
		FROM users
		WHERE name = ? AND active = 1`, name)

	u := &User{}
	var sid sql.NullString
	var tsJoin, lastUpdate string
	err := row.Scan(&u.ID, &u.Name, &u.Pubkey, &u.FP, &u.Token, &sid, &tsJoin, &lastUpdate, &u.Active)
	if err != nil {
		return nil, err
	}
	if sid.Valid {
		u.SID = &sid.String
	}
	u.TsJoin, _ = parseTime(tsJoin)
	u.LastUpdate, _ = parseTime(lastUpdate)
	return u, nil
}

// GetUserByID looks up a user by ID.
func (s *Storage) GetUserByID(id int64) (*User, error) {
	row := s.db.QueryRow(`
		SELECT id, name, pubkey, fp, token, sid, ts_join, last_update, active
		FROM users
		WHERE id = ?`, id)

	u := &User{}
	var sid sql.NullString
	var tsJoin, lastUpdate string
	err := row.Scan(&u.ID, &u.Name, &u.Pubkey, &u.FP, &u.Token, &sid, &tsJoin, &lastUpdate, &u.Active)
	if err != nil {
		return nil, err
	}
	if sid.Valid {
		u.SID = &sid.String
	}
	u.TsJoin, _ = parseTime(tsJoin)
	u.LastUpdate, _ = parseTime(lastUpdate)
	return u, nil
}

// CreateUser inserts a new user into the database.
func (s *Storage) CreateUser(name, pubkey, fp, token string) (*User, error) {
	now := time.Now().UTC()
	res, err := s.db.Exec(`
		INSERT INTO users (name, pubkey, fp, token, ts_join, last_update, active)
		VALUES (?, ?, ?, ?, ?, ?, 1)`,
		name, pubkey, fp, token, now, now)
	if err != nil {
		return nil, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}

	return &User{
		ID:         id,
		Name:       name,
		Pubkey:     pubkey,
		FP:         fp,
		Token:      token,
		TsJoin:     now,
		LastUpdate: now,
		Active:     true,
	}, nil
}

// UpdateUserStatus updates online status and session ID.
func (s *Storage) UpdateUserStatus(id int64, active bool, sid *string) error {
	now := time.Now().UTC()
	var sVal sql.NullString
	if sid != nil {
		sVal = sql.NullString{String: *sid, Valid: true}
	}
	_, err := s.db.Exec(`
		UPDATE users
		SET active = ?, sid = ?, last_update = ?
		WHERE id = ?`,
		active, sVal, now, id)
	return err
}

// GetActiveUsers returns all users (ordered by active status and last_update) along with their last message received by current user.
func (s *Storage) GetActiveUsers(excludeUserID int64) ([]UserWithLastMsg, error) {
	rows, err := s.db.Query(`
		SELECT u.id, u.name, u.fp, u.pubkey, u.last_update, u.active,
		       (SELECT m.msg FROM user_messages m
		        WHERE m.sender_id = u.id AND m.recipient_id = ?
		        ORDER BY m.id DESC LIMIT 1) as msg
		FROM users u
		WHERE u.id != ?
		ORDER BY u.active DESC, u.last_update DESC`,
		excludeUserID, excludeUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []UserWithLastMsg
	for rows.Next() {
		var u UserWithLastMsg
		var msg sql.NullString
		var lastUpdate string
		if err := rows.Scan(&u.ID, &u.Name, &u.FP, &u.Pubkey, &lastUpdate, &u.Active, &msg); err != nil {
			return nil, err
		}
		u.LastUpdate = lastUpdate
		if msg.Valid {
			u.Msg = &msg.String
		}
		result = append(result, u)
	}

	if result == nil {
		result = []UserWithLastMsg{}
	}
	return result, nil
}

// SaveUserMessage saves a direct message between two users.
func (s *Storage) SaveUserMessage(senderID, recipientID int64, msg string) (*Message, error) {
	now := time.Now().UTC()
	seqRes, err := s.db.Exec("INSERT INTO message_sequence DEFAULT VALUES")
	if err != nil {
		return nil, err
	}
	id, err := seqRes.LastInsertId()
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO user_messages (id, sender_id, recipient_id, msg, ts)
		VALUES (?, ?, ?, ?, ?)`,
		id, senderID, recipientID, msg, now)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:        id,
		Sender:    senderID,
		Recipient: recipientID,
		Msg:       msg,
		TS:        now,
		IsGroup:   false,
	}, nil
}

// SaveGroupMessage saves a message directed to the group.
func (s *Storage) SaveGroupMessage(senderID int64, msg string) (*Message, error) {
	now := time.Now().UTC()
	seqRes, err := s.db.Exec("INSERT INTO message_sequence DEFAULT VALUES")
	if err != nil {
		return nil, err
	}
	id, err := seqRes.LastInsertId()
	if err != nil {
		return nil, err
	}

	_, err = s.db.Exec(`
		INSERT INTO group_messages (id, sender_id, msg, ts)
		VALUES (?, ?, ?, ?)`,
		id, senderID, msg, now)
	if err != nil {
		return nil, err
	}

	return &Message{
		ID:        id,
		Sender:    senderID,
		Recipient: "default",
		Msg:       msg,
		TS:        now,
		IsGroup:   true,
	}, nil
}

// GetUserMessages gets conversation between user A and user B.
func (s *Storage) GetUserMessages(userA, userB int64) ([]Message, error) {
	rows, err := s.db.Query(`
		SELECT id, sender_id, recipient_id, msg, ts
		FROM user_messages
		WHERE (sender_id = ? AND recipient_id = ?)
		   OR (sender_id = ? AND recipient_id = ?)
		ORDER BY id ASC`,
		userA, userB, userB, userA)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Message
	for rows.Next() {
		var m Message
		var senderID, recipientID sql.NullInt64
		var tsStr string
		if err := rows.Scan(&m.ID, &senderID, &recipientID, &m.Msg, &tsStr); err != nil {
			return nil, err
		}
		if senderID.Valid {
			m.Sender = senderID.Int64
		}
		if recipientID.Valid {
			m.Recipient = recipientID.Int64
		}
		m.TS, _ = parseTime(tsStr)
		m.IsGroup = false
		result = append(result, m)
	}

	if result == nil {
		result = []Message{}
	}
	return result, nil
}

// GetGroupMessages gets latest messages for group default.
func (s *Storage) GetGroupMessages(userJoin time.Time, limit int) ([]Message, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.Query(`
		SELECT gm.id, gm.sender_id, gm.msg, gm.ts
		FROM group_messages gm
		ORDER BY gm.id DESC
		LIMIT ?`,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []Message
	for rows.Next() {
		var m Message
		var senderID sql.NullInt64
		var tsStr string
		if err := rows.Scan(&m.ID, &senderID, &m.Msg, &tsStr); err != nil {
			return nil, err
		}
		if senderID.Valid {
			m.Sender = senderID.Int64
		}
		m.Recipient = "default"
		m.TS, _ = parseTime(tsStr)
		m.IsGroup = true
		result = append(result, m)
	}

	// Reverse to order by ID ascending
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	if result == nil {
		result = []Message{}
	}
	return result, nil
}

// SyncMessages implements cursor synchronization for Tor fallback.
func (s *Storage) SyncMessages(userID int64, sinceID int64, groupID string) ([]Message, int64, error) {
	var messages []Message
	maxID := sinceID

	// 1. Direct messages where user is sender or recipient
	dmRows, err := s.db.Query(`
		SELECT id, sender_id, recipient_id, msg, ts
		FROM user_messages
		WHERE id > ? AND (sender_id = ? OR recipient_id = ?)
		ORDER BY id ASC`,
		sinceID, userID, userID)
	if err == nil {
		defer dmRows.Close()
		for dmRows.Next() {
			var m Message
			var senderID, recipientID sql.NullInt64
			var tsStr string
			if err := dmRows.Scan(&m.ID, &senderID, &recipientID, &m.Msg, &tsStr); err == nil {
				if senderID.Valid {
					m.Sender = senderID.Int64
				}
				if recipientID.Valid {
					m.Recipient = recipientID.Int64
				}
				m.TS, _ = parseTime(tsStr)
				m.IsGroup = false
				messages = append(messages, m)
				if m.ID > maxID {
					maxID = m.ID
				}
			}
		}
	}

	// 2. Group messages if requested (e.g. "default")
	if groupID != "" {
		gmRows, err := s.db.Query(`
			SELECT gm.id, gm.sender_id, gm.msg, gm.ts
			FROM group_messages gm
			JOIN users u ON u.id = gm.sender_id
			WHERE gm.id > ? AND u.active = 1
			ORDER BY gm.id ASC`,
			sinceID)
		if err == nil {
			defer gmRows.Close()
			for gmRows.Next() {
				var m Message
				var senderID sql.NullInt64
				var tsStr string
				if err := gmRows.Scan(&m.ID, &senderID, &m.Msg, &tsStr); err == nil {
					if senderID.Valid {
						m.Sender = senderID.Int64
					}
					m.Recipient = "default"
					m.TS, _ = parseTime(tsStr)
					m.IsGroup = true
					messages = append(messages, m)
					if m.ID > maxID {
						maxID = m.ID
					}
				}
			}
		}
	}

	// Sort all messages by ID
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].ID < messages[j].ID
	})

	if messages == nil {
		messages = []Message{}
	}

	return messages, maxID, nil
}

func parseTime(tsStr string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, tsStr); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unable to parse time: %s", tsStr)
}
