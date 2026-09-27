package leetgrinder

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// APITokenPrefix marks personal API tokens used by the browser extension.
const APITokenPrefix = "lg_"

// apiTokenLength is the prefix plus 32 random bytes in unpadded base64url.
var apiTokenLength = len(APITokenPrefix) + base64.RawURLEncoding.EncodedLen(32)

// APIToken is a stored token's metadata. The token itself is never stored;
// only its SHA-256 hash is.
type APIToken struct {
	ID         string
	Name       string
	CreatedAt  time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
}

func (t APIToken) Revoked() bool { return t.RevokedAt != nil }

// NewAPIToken returns a fresh random token and the hash to store for it.
func NewAPIToken() (token string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err = rand.Read(b); err != nil {
		return "", nil, err
	}
	token = APITokenPrefix + base64.RawURLEncoding.EncodeToString(b)
	return token, HashAPIToken(token), nil
}

// HashAPIToken is the stored form of a token.
func HashAPIToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// WellFormedAPIToken rejects values that could never be issued tokens, so
// junk never reaches the database.
func WellFormedAPIToken(token string) bool {
	if len(token) != apiTokenLength || !strings.HasPrefix(token, APITokenPrefix) {
		return false
	}
	_, err := base64.RawURLEncoding.DecodeString(token[len(APITokenPrefix):])
	return err == nil
}

// ValidateAPITokenName trims name and checks it is a short, printable label.
func ValidateAPITokenName(name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" || !utf8.ValidString(name) || utf8.RuneCountInString(name) > 64 {
		return "", errors.New("name the token in 1 to 64 characters")
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return "", errors.New("token names cannot contain control characters")
		}
	}
	return name, nil
}

// APITokensSection is the settings page's token-management state.
type APITokensSection struct {
	Tokens []APIToken
	// Created is the plaintext of a just-created token. It is rendered only
	// in the response to the create request and never again.
	Created     string
	CreatedName string
	// Name keeps a rejected draft name.
	Name  string
	Error string
	// Unavailable is set when the token list could not be loaded.
	Unavailable bool
}

// ActiveCount counts tokens that have not been revoked.
func (s APITokensSection) ActiveCount() int {
	n := 0
	for _, t := range s.Tokens {
		if !t.Revoked() {
			n++
		}
	}
	return n
}

func TokenTimeLabel(t *time.Time, loc *time.Location) string {
	if t == nil {
		return "Never"
	}
	return t.In(loc).Format("Mon 2 Jan 2006 15:04")
}
