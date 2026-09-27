package leetgrinder

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
)

// ErrNoSecretKey means LEETGRINDER_SECRET_KEY is unset in production.
var ErrNoSecretKey = errors.New("LEETGRINDER_SECRET_KEY is not configured")

// SecretBox encrypts small secrets, such as the ntfy token, with AES-256-GCM.
// Ciphertexts are nonce-prefixed.
type SecretBox struct{ aead cipher.AEAD }

// NewSecretBox takes the 32-byte key from config.Config.LeetgrinderSecretKey.
func NewSecretBox(key []byte) (*SecretBox, error) {
	if len(key) == 0 {
		return nil, ErrNoSecretKey
	}
	if len(key) != 32 {
		return nil, errors.New("leetgrinder secret key must be 32 bytes")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &SecretBox{aead: aead}, nil
}

func (b *SecretBox) Seal(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, b.aead.NonceSize(), b.aead.NonceSize()+len(plaintext)+b.aead.Overhead())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return b.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (b *SecretBox) Open(ciphertext []byte) ([]byte, error) {
	n := b.aead.NonceSize()
	if len(ciphertext) < n+b.aead.Overhead() {
		return nil, errors.New("ciphertext is too short")
	}
	return b.aead.Open(nil, ciphertext[:n], ciphertext[n:], nil)
}
