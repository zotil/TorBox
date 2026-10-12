package server

import (
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/armor"
	"torbox/chatsecure/storage"
)

type KeyInfo struct {
	Fingerprint string
	Name        string
	Email       string
}

// GenerateSecretKey creates a secure random hex key.
func GenerateSecretKey() string {
	b := make([]byte, 64)
	if _, err := rand.Read(b); err != nil {
		return "torbox-chatsecure-fallback-secret-key"
	}
	return hex.EncodeToString(b)
}

// ParsePGPPublicKey extracts the fingerprint and identity name from an armored PGP public key.
// It supports all OpenPGP algorithms (including RSA, DSA, ECDSA, and EdDSA/Curve25519/algo 22).
func ParsePGPPublicKey(armoredKey string) (*KeyInfo, error) {
	// First attempt using standard openpgp library
	info, err := parseWithOpenPGPLib(armoredKey)
	if err == nil && info != nil && info.Fingerprint != "" {
		return info, nil
	}

	// Fallback to direct RFC 4880 / RFC 9580 packet parser (supports Ed25519/Curve25519 and newer algorithms)
	return parseOpenPGPPackets(armoredKey)
}

func parseWithOpenPGPLib(armoredKey string) (*KeyInfo, error) {
	reader := strings.NewReader(armoredKey)
	entityList, err := openpgp.ReadArmoredKeyRing(reader)
	if err != nil {
		return nil, err
	}
	if len(entityList) == 0 {
		return nil, errors.New("no PGP entities found in public key")
	}

	entity := entityList[0]
	fp := fmt.Sprintf("%X", entity.PrimaryKey.Fingerprint)

	var name, email string
	for _, ident := range entity.Identities {
		if ident.UserId != nil {
			if ident.UserId.Name != "" {
				name = ident.UserId.Name
			}
			if ident.UserId.Email != "" {
				email = ident.UserId.Email
			}
			if name != "" {
				break
			}
			if ident.UserId.Id != "" {
				raw := ident.UserId.Id
				if idx := strings.Index(raw, "<"); idx > 0 {
					name = strings.TrimSpace(raw[:idx])
				} else {
					name = strings.TrimSpace(raw)
				}
				if name != "" {
					break
				}
			}
		}
	}

	return &KeyInfo{
		Fingerprint: fp,
		Name:        name,
		Email:       email,
	}, nil
}

// parseOpenPGPPackets parses raw OpenPGP packets to compute RFC 4880/RFC 9580 fingerprints
// for any public key algorithm (including EdDSA/Ed25519/Curve25519 type 22).
func parseOpenPGPPackets(armoredKey string) (*KeyInfo, error) {
	block, err := armor.Decode(strings.NewReader(armoredKey))
	if err != nil {
		return nil, fmt.Errorf("failed to decode armored PGP key: %w", err)
	}
	if block.Type != openpgp.PublicKeyType {
		return nil, fmt.Errorf("expected PGP public key block, got: %s", block.Type)
	}

	data, err := io.ReadAll(block.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read armored packet data: %w", err)
	}

	var pubKeyBody []byte
	var rawUserID string
	offset := 0

	for offset < len(data) {
		if data[offset]&0x80 == 0 {
			return nil, errors.New("invalid packet header: bit 7 not set")
		}

		isNewFormat := (data[offset] & 0x40) != 0
		var tag byte
		var bodyLen int

		if isNewFormat {
			tag = data[offset] & 0x3F
			offset++
			if offset >= len(data) {
				return nil, errors.New("unexpected EOF in packet header")
			}
			firstLen := data[offset]
			if firstLen < 192 {
				bodyLen = int(firstLen)
				offset++
			} else if firstLen < 224 {
				if offset+1 >= len(data) {
					return nil, errors.New("unexpected EOF in 2-byte length")
				}
				bodyLen = (int(firstLen-192) << 8) + int(data[offset+1]) + 192
				offset += 2
			} else if firstLen == 255 {
				if offset+4 >= len(data) {
					return nil, errors.New("unexpected EOF in 4-byte length")
				}
				bodyLen = int(binary.BigEndian.Uint32(data[offset+1 : offset+5]))
				offset += 5
			} else {
				// Partial body length
				bodyLen = 1 << (firstLen & 0x1F)
				offset++
			}
		} else {
			tag = (data[offset] >> 2) & 0x0F
			lenType := data[offset] & 0x03
			offset++
			switch lenType {
			case 0:
				if offset >= len(data) {
					return nil, errors.New("unexpected EOF in old 1-byte length")
				}
				bodyLen = int(data[offset])
				offset++
			case 1:
				if offset+1 >= len(data) {
					return nil, errors.New("unexpected EOF in old 2-byte length")
				}
				bodyLen = int(binary.BigEndian.Uint16(data[offset : offset+2]))
				offset += 2
			case 2:
				if offset+3 >= len(data) {
					return nil, errors.New("unexpected EOF in old 4-byte length")
				}
				bodyLen = int(binary.BigEndian.Uint32(data[offset : offset+4]))
				offset += 4
			case 3:
				bodyLen = len(data) - offset
			}
		}

		if offset+bodyLen > len(data) {
			return nil, errors.New("packet length exceeds available data")
		}

		body := data[offset : offset+bodyLen]
		offset += bodyLen

		// Tag 6 = Public Key Packet
		if tag == 6 && len(pubKeyBody) == 0 {
			pubKeyBody = body
		} else if tag == 13 && rawUserID == "" {
			// Tag 13 = User ID Packet
			rawUserID = string(body)
		}
	}

	if len(pubKeyBody) == 0 {
		return nil, errors.New("no public key packet (tag 6) found in PGP block")
	}

	var fp string
	version := pubKeyBody[0]

	if version == 4 {
		// RFC 4880 Section 12.2: SHA-1 of 0x99 || 2-octet length || body
		h := sha1.New()
		h.Write([]byte{0x99, byte(len(pubKeyBody) >> 8), byte(len(pubKeyBody))})
		h.Write(pubKeyBody)
		fp = fmt.Sprintf("%X", h.Sum(nil))
	} else if version == 5 {
		// RFC 9580 Section 11.1: SHA-256 of 0x9A || 4-octet length || body
		h := sha256.New()
		lenBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(lenBytes, uint32(len(pubKeyBody)))
		h.Write([]byte{0x9A})
		h.Write(lenBytes)
		h.Write(pubKeyBody)
		fp = fmt.Sprintf("%X", h.Sum(nil))
	} else {
		return nil, fmt.Errorf("unsupported PGP public key version: %d", version)
	}

	name := rawUserID
	email := ""
	if idx := strings.Index(rawUserID, "<"); idx >= 0 {
		name = strings.TrimSpace(rawUserID[:idx])
		rest := rawUserID[idx+1:]
		if endIdx := strings.Index(rest, ">"); endIdx >= 0 {
			email = strings.TrimSpace(rest[:endIdx])
		}
	}

	return &KeyInfo{
		Fingerprint: fp,
		Name:        name,
		Email:       email,
	}, nil
}

// GenerateToken creates sha1(fp + secretKey) as used by ChatSecure.
func GenerateToken(fp, secretKey string) string {
	h := sha1.New()
	h.Write([]byte(fp + secretKey))
	return hex.EncodeToString(h.Sum(nil))
}

// ExtractToken extracts the token from Authorization header or query parameter.
func ExtractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 {
			prefix := strings.ToLower(parts[0])
			if prefix == "token" || prefix == "bearer" {
				return strings.TrimSpace(parts[1])
			}
		}
		// If raw token was passed
		return strings.TrimSpace(authHeader)
	}

	// Also check query param ?token=
	if qToken := r.URL.Query().Get("token"); qToken != "" {
		return qToken
	}

	// Also check cookie tcs_auth
	if c, err := r.Cookie("tcs_auth"); err == nil && c.Value != "" {
		val, _ := url.QueryUnescape(c.Value)
		var authCookie struct {
			Token string `json:"token"`
		}
		if err := json.Unmarshal([]byte(val), &authCookie); err == nil && authCookie.Token != "" {
			return authCookie.Token
		}
	}

	return ""
}

// AuthenticateRequest validates the token against the database and returns the User.
func AuthenticateRequest(r *http.Request, s *storage.Storage) (*storage.User, error) {
	token := ExtractToken(r)
	if token == "" {
		return nil, errors.New("missing authorization token")
	}

	user, err := s.GetUserByToken(token)
	if err != nil || user == nil {
		return nil, errors.New("invalid or expired token")
	}

	return user, nil
}
