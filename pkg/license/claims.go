package license

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	ErrLicenseNil                 = errors.New("license is nil")
	ErrLicenseNotFound            = errors.New("license not found")
	ErrLicenseInvalidFormat       = errors.New("invalid license format")
	ErrLicenseInvalidSignature    = errors.New("license signature verification failed")
	ErrLicenseExpired             = errors.New("license has expired")
	ErrLicenseNotYetValid         = errors.New("license is not active yet")
	ErrLicenseFingerprintMismatch = errors.New("hardware fingerprint mismatch")
	ErrLicenseDeviceLimitExceeded = errors.New("device quota limit exceeded")
)

// LicenseClaims 授权声明载荷
type LicenseClaims struct {
	LicenseId    string            `json:"licenseId"`             // 授权唯一标识 (UUID)
	CustomerName string            `json:"customerName"`          // 客户/企业名称
	IssuedAt     int64             `json:"issuedAt"`              // 签发时间戳 (Unix秒)
	NotBefore    int64             `json:"notBefore"`             // 生效时间戳 (Unix秒)
	ExpiresAt    int64             `json:"expiresAt"`             // 到期时间戳 (Unix秒，0 表示永久)
	MaxDevices   int64             `json:"maxDevices"`            // 最大允许接入设备数 (0 为无限制)
	Fingerprint  string            `json:"fingerprint,omitempty"` // 绑定的机器指纹哈希 (为空则不限制机器)
	Extra        map[string]string `json:"extra,omitempty"`       // 预留自定义扩展参数
}

// LicenseEnvelope 许可证信封（签名包装）
type LicenseEnvelope struct {
	Payload   string `json:"payload"`   // base64(json(LicenseClaims))
	Signature string `json:"signature"` // base64(signature)
	Algorithm string `json:"algorithm"` // 签名算法，例如 "ed25519"
	Version   int    `json:"version"`   // 格式版本号，当前为 1
}

// FormatExpiresAt 格式化到期时间
func (c *LicenseClaims) FormatExpiresAt() string {
	if c.ExpiresAt <= 0 {
		return "Permanent (永久有效)"
	}
	return time.Unix(c.ExpiresAt, 0).Format("2006-01-02 15:04:05")
}

// FormatIssuedAt 格式化签发时间
func (c *LicenseClaims) FormatIssuedAt() string {
	if c.IssuedAt <= 0 {
		return "-"
	}
	return time.Unix(c.IssuedAt, 0).Format("2006-01-02 15:04:05")
}

// IsPermanent 是否永久
func (c *LicenseClaims) IsPermanent() bool {
	return c.ExpiresAt <= 0
}

// Validate 基础校验
func (c *LicenseClaims) Validate(currentFingerprint string, now time.Time) error {
	if c == nil {
		return ErrLicenseNil
	}
	nowSec := now.Unix()
	if c.NotBefore > 0 && nowSec < c.NotBefore {
		return fmt.Errorf("%w: active from %s", ErrLicenseNotYetValid, time.Unix(c.NotBefore, 0).Format("2006-01-02 15:04:05"))
	}
	if c.ExpiresAt > 0 && nowSec > c.ExpiresAt {
		return fmt.Errorf("%w: expired at %s", ErrLicenseExpired, time.Unix(c.ExpiresAt, 0).Format("2006-01-02 15:04:05"))
	}
	// 指纹校验
	if len(strings.TrimSpace(c.Fingerprint)) > 0 {
		targetFp := NormalizeFingerprint(c.Fingerprint)
		currFp := NormalizeFingerprint(currentFingerprint)
		if targetFp != currFp {
			return fmt.Errorf("%w: expected %s, current is %s", ErrLicenseFingerprintMismatch, targetFp, currFp)
		}
	}
	return nil
}

// DecodePayload 解析载荷
func DecodePayload(payloadBase64 string) (*LicenseClaims, error) {
	data, err := base64.StdEncoding.DecodeString(payloadBase64)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLicenseInvalidFormat, err)
	}
	var claims LicenseClaims
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrLicenseInvalidFormat, err)
	}
	return &claims, nil
}

// VerifyEnvelope 验签
func VerifyEnvelope(env *LicenseEnvelope, pubKey ed25519.PublicKey) (*LicenseClaims, error) {
	if env == nil {
		return nil, ErrLicenseNil
	}
	if len(pubKey) == 0 {
		return nil, errors.New("public key is empty")
	}
	sigBytes, err := base64.StdEncoding.DecodeString(env.Signature)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid signature encoding", ErrLicenseInvalidSignature)
	}
	if !ed25519.Verify(pubKey, []byte(env.Payload), sigBytes) {
		return nil, ErrLicenseInvalidSignature
	}
	return DecodePayload(env.Payload)
}

// ParsePublicKey 从 PEM 字符串或文件加载 Ed25519 公钥
func ParsePublicKey(keyOrPath string) (ed25519.PublicKey, error) {
	raw := strings.TrimSpace(keyOrPath)
	if len(raw) == 0 {
		raw = DefaultPublicKeyPEM
	}

	var pemBytes []byte
	if strings.Contains(raw, "-----BEGIN") {
		pemBytes = []byte(raw)
	} else if _, err := os.Stat(raw); err == nil {
		data, err := os.ReadFile(raw)
		if err != nil {
			return nil, fmt.Errorf("read public key file failed: %w", err)
		}
		pemBytes = data
	} else {
		// 尝试作为 Base64 或使用内置
		pemBytes = []byte(DefaultPublicKeyPEM)
	}

	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode public key PEM block")
	}

	key, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse pkix public key failed: %w", err)
	}
	edKey, ok := key.(ed25519.PublicKey)
	if !ok {
		return nil, errors.New("key is not ed25519 public key")
	}
	return edKey, nil
}
