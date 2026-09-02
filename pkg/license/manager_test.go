package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLicenseVerificationAndClaims(t *testing.T) {
	pub, pri, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	now := time.Now()
	claims := &LicenseClaims{
		LicenseId:    "test-id-001",
		CustomerName: "测试企业",
		IssuedAt:     now.Unix(),
		NotBefore:    now.Unix() - 5,
		ExpiresAt:    now.Unix() + 86400*30,
		MaxDevices:   500,
		Fingerprint:  "A1B2-C3D4-E5F6-7890",
	}

	payload, err := json.Marshal(claims)
	require.NoError(t, err)

	payloadB64 := base64.StdEncoding.EncodeToString(payload)
	sig := ed25519.Sign(pri, []byte(payloadB64))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	env := &LicenseEnvelope{
		Payload:   payloadB64,
		Signature: sigB64,
		Algorithm: "ed25519",
		Version:   1,
	}

	verified, err := VerifyEnvelope(env, pub)
	require.NoError(t, err)
	require.Equal(t, claims.CustomerName, verified.CustomerName)
	require.Equal(t, claims.MaxDevices, verified.MaxDevices)

	// Fingerprint validation
	err = verified.Validate("a1b2-c3d4-e5f6-7890", now)
	require.NoError(t, err)

	err = verified.Validate("DIFF-FP", now)
	require.ErrorIs(t, err, ErrLicenseFingerprintMismatch)
}

