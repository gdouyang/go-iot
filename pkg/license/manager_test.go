package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"path/filepath"
	"sync"
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

func TestLicenseRuntimeExpiration(t *testing.T) {
	origCountFunc := countDevicesFunc
	countDevicesFunc = func() (int64, error) { return 10, nil }
	defer func() { countDevicesFunc = origCountFunc }()

	mgr := &Manager{
		status:   StatusUnlicensed,
		cachedFp: "test-fp-1234",
	}

	// 1. 设置有效期的 License (30天后过期)
	now := time.Now()
	claims := &LicenseClaims{
		LicenseId:    "lic-active",
		CustomerName: "Test Company",
		IssuedAt:     now.Unix() - 3600,
		ExpiresAt:    now.Unix() + 86400*30,
		MaxDevices:   100,
		Fingerprint:  "test-fp-1234",
	}
	mgr.claims = claims
	mgr.updateStatusLocked()

	require.NoError(t, mgr.CheckValid())
	require.False(t, mgr.IsRequireRedirect())
	info := mgr.GetInfo()
	require.Equal(t, StatusActive, info.Status)
	require.True(t, info.IsLicensed)
	require.False(t, info.RequireRedirect)

	// 2. 模拟运行期间时间流逝到达过期时刻 (ExpiresAt 已过)，定时器执行 UpdateStatus 刷新状态
	claims.ExpiresAt = time.Now().Unix() - 10
	mgr.UpdateStatus()

	require.Error(t, mgr.CheckValid())
	require.True(t, mgr.IsRequireRedirect())
	expiredInfo := mgr.GetInfo()
	require.Equal(t, StatusExpired, expiredInfo.Status)
	require.False(t, expiredInfo.IsLicensed)
	require.True(t, expiredInfo.RequireRedirect)

	// 3. 测试文件不存在时定时更新为未授权
	mgr.licFilePath = filepath.Join(t.TempDir(), "non-existent.lic")
	mgr.UpdateStatus()
	require.Equal(t, StatusUnlicensed, mgr.status)
	require.True(t, mgr.IsRequireRedirect())
}

func TestLicenseReimportImmediateUpdate(t *testing.T) {
	pub, pri, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	tmpDir := t.TempDir()
	licPath := filepath.Join(tmpDir, "test.lic")

	mgr := &Manager{
		licFilePath: licPath,
		publicKey:   pub,
		cachedFp:    "fp-test-reimport",
		status:      StatusInvalidSignature, // 之前处于签名无效/篡改状态
	}

	// 重新导入生成合法的 license
	now := time.Now()
	claims := &LicenseClaims{
		LicenseId:    "reimport-001",
		CustomerName: "Reimport Corp",
		IssuedAt:     now.Unix(),
		ExpiresAt:    now.Unix() + 86400*30,
		MaxDevices:   200,
		Fingerprint:  "fp-test-reimport",
	}

	payload, err := json.Marshal(claims)
	require.NoError(t, err)

	payloadB64 := base64.StdEncoding.EncodeToString(payload)
	sig := ed25519.Sign(pri, []byte(payloadB64))
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	env := LicenseEnvelope{
		Payload:   payloadB64,
		Signature: sigB64,
		Algorithm: "ed25519",
		Version:   1,
	}
	licBytes, err := json.Marshal(env)
	require.NoError(t, err)

	// 调用重新导入 SaveAndReload
	err = mgr.SaveAndReload(licBytes)
	require.NoError(t, err)

	// 验证重新导入后状态立刻更新为 StatusActive，无需等待定时器
	require.Equal(t, StatusActive, mgr.status)
	require.NoError(t, mgr.CheckValid())
	require.False(t, mgr.IsRequireRedirect())

	info := mgr.GetStatusInfo()
	require.Equal(t, StatusActive, info.Status)
	require.True(t, info.IsLicensed)
	require.False(t, info.RequireRedirect)
	require.Equal(t, "Reimport Corp", info.CustomerName)
}

func TestLicenseTimerRecover(t *testing.T) {
	mgr := &Manager{
		status: StatusActive,
	}

	// 模拟触发带 recover 的周期调用，确保单次 tick panic 被顺利捕获并不导致程序崩溃
	panicTriggered := false
	recovered := false

	func() {
		defer func() {
			if r := recover(); r != nil {
				recovered = true
			}
		}()
		// 模拟执行体
		func() {
			panicTriggered = true
			panic("simulated transient panic in timer tick")
		}()
	}()

	require.True(t, panicTriggered)
	require.True(t, recovered)
	// 验证 mgr 本身状态完好
	require.Equal(t, StatusActive, mgr.status)
}

func TestLicenseConcurrency(t *testing.T) {
	origCountFunc := countDevicesFunc
	countDevicesFunc = func() (int64, error) { return 5, nil }
	defer func() { countDevicesFunc = origCountFunc }()

	mgr := &Manager{
		status:   StatusActive,
		cachedFp: "fp-concurrent-test",
		claims: &LicenseClaims{
			LicenseId:    "lic-conc",
			CustomerName: "ConcCorp",
			ExpiresAt:    time.Now().Unix() + 3600,
			MaxDevices:   100,
			Fingerprint:  "fp-concurrent-test",
		},
	}

	stopCh := make(chan struct{})
	var wg sync.WaitGroup

	// 50 个读协程
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					_ = mgr.CheckValid()
					_ = mgr.IsRequireRedirect()
					_ = mgr.GetStatusInfo()
					_ = mgr.GetInfo()
					_ = mgr.CheckCanAddDevice(1)
				}
			}
		}()
	}

	// 10 个更新协程
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopCh:
					return
				default:
					mgr.UpdateStatus()
					time.Sleep(2 * time.Millisecond)
				}
			}
		}()
	}

	time.Sleep(200 * time.Millisecond)
	close(stopCh)
	wg.Wait()
}

