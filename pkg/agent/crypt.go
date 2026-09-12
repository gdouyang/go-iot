package agent

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

const encPrefix = "enc:v1:"

const builtinSecret = "goiot-agent-builtin-secret-key-v1"

func secretKey() []byte {
	sum := sha256.Sum256([]byte(builtinSecret))
	return sum[:]
}

func SealAPIKey(plain string) (string, error) {
	plain = strings.TrimSpace(plain)
	if plain == "" {
		return "", nil
	}
	if strings.HasPrefix(plain, encPrefix) {
		return plain, nil
	}
	block, err := aes.NewCipher(secretKey())
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(plain), nil)
	return encPrefix + base64.StdEncoding.EncodeToString(out), nil
}

func RevealAPIKey(stored string) string {
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return ""
	}
	if !strings.HasPrefix(stored, encPrefix) {
		return stored
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(stored, encPrefix))
	if err != nil {
		return ""
	}
	block, err := aes.NewCipher(secretKey())
	if err != nil {
		return ""
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return ""
	}
	ns := gcm.NonceSize()
	if len(raw) < ns {
		return ""
	}
	plain, err := gcm.Open(nil, raw[:ns], raw[ns:], nil)
	if err != nil {
		return ""
	}
	return string(plain)
}

func IsSealedKey(stored string) bool {
	return strings.HasPrefix(strings.TrimSpace(stored), encPrefix)
}

func MustSealAPIKey(plain string) string {
	s, err := SealAPIKey(plain)
	if err != nil {
		return plain
	}
	return s
}

func keySealError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: encrypt api key: %v", ReasonInvalidArgs, err)
}
