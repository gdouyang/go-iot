package license

import (
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go-iot/pkg/es/orm"
	logs "go-iot/pkg/logger"
	"go-iot/pkg/models"
	"go-iot/pkg/option"
)

const (
	StatusUnlicensed          = "unlicensed"
	StatusActive              = "active"
	StatusExpiringSoon        = "expiring_soon"
	StatusExpired             = "expired"
	StatusFingerprintMismatch = "fingerprint_mismatch"
	StatusInvalidSignature    = "invalid_signature"
)

// LicenseInfo 返回给 API 与前端的授权视图对象
type LicenseInfo struct {
	Status             string   `json:"status"` // unlicensed, active, expiring_soon, expired, fingerprint_mismatch, invalid_signature
	StatusText         string   `json:"statusText"`
	CustomerName       string   `json:"customerName"`
	LicenseId          string   `json:"licenseId"`
	IssuedAt           int64    `json:"issuedAt"`
	IssuedAtFormatted  string   `json:"issuedAtFormatted"`
	ExpiresAt          int64    `json:"expiresAt"`
	ExpiresAtFormatted string `json:"expiresAtFormatted"`
	MaxDevices         int64  `json:"maxDevices"`
	CurrentDevices     int64  `json:"currentDevices"`
	Fingerprint        string `json:"fingerprint"`        // License 绑定的指纹
	MachineFingerprint string `json:"machineFingerprint"` // 当前机器的实际指纹
	IsLicensed         bool   `json:"isLicensed"`
	RequireRedirect    bool   `json:"requireRedirect"`
}

type Manager struct {
	mu          sync.RWMutex
	licFilePath string
	publicKey   ed25519.PublicKey
	claims      *LicenseClaims
	status      string
	lastErr     error
	cachedFp    string
}

var defaultManager = &Manager{
	licFilePath: "conf/license.lic",
	status:      StatusUnlicensed,
}

// Default 返回全局单例 Manager
func Default() *Manager {
	return defaultManager
}

// SetTestActive 用于单元测试中临时将 License 设置为激活状态
func SetTestActive(claims *LicenseClaims) func() {
	defaultManager.mu.Lock()
	oldClaims := defaultManager.claims
	oldStatus := defaultManager.status
	if claims == nil {
		claims = &LicenseClaims{
			LicenseId:    "test-active",
			CustomerName: "Test",
			MaxDevices:   0,
		}
	}
	defaultManager.claims = claims
	defaultManager.status = StatusActive
	defaultManager.mu.Unlock()

	return func() {
		defaultManager.mu.Lock()
		defaultManager.claims = oldClaims
		defaultManager.status = oldStatus
		defaultManager.mu.Unlock()
	}
}

// Init 初始化 License 校验器
func Init(opt *option.Options) {
	defaultManager.mu.Lock()
	defer defaultManager.mu.Unlock()

	if opt != nil && len(opt.License.LicFile) > 0 {
		defaultManager.licFilePath = opt.License.LicFile
	} else {
		defaultManager.licFilePath = "conf/license.lic"
	}

	pubKey, err := ParsePublicKey(opt.License.PublicKey)
	if err != nil {
		logs.Errorf("license: parse public key failed: %v, using default builtin public key", err)
		pubKey, _ = ParsePublicKey(DefaultPublicKeyPEM)
	}
	defaultManager.publicKey = pubKey

	// 采集机器指纹
	if fp, err := GetMachineFingerprint(); err == nil {
		defaultManager.cachedFp = fp
	}

	// 初始加载
	defaultManager.loadLocked()
}

// Reload 重新从文件加载
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.loadLocked()
}

// SaveAndReload 保存上传的 .lic 内容并立即生效
func (m *Manager) SaveAndReload(licBytes []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var env LicenseEnvelope
	if err := json.Unmarshal(licBytes, &env); err != nil {
		return fmt.Errorf("%w: invalid JSON envelope", ErrLicenseInvalidFormat)
	}

	claims, err := VerifyEnvelope(&env, m.publicKey)
	if err != nil {
		return fmt.Errorf("license 验签失败: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(m.licFilePath), 0755); err != nil {
		return fmt.Errorf("create license dir failed: %w", err)
	}

	if err := os.WriteFile(m.licFilePath, licBytes, 0644); err != nil {
		return fmt.Errorf("write license file failed: %w", err)
	}

	m.claims = claims
	m.updateStatusLocked()
	logs.Infof("license: uploaded and activated successfully for customer %s (max devices: %d, expires: %s)",
		claims.CustomerName, claims.MaxDevices, claims.FormatExpiresAt())
	return nil
}

func (m *Manager) loadLocked() error {
	if _, err := os.Stat(m.licFilePath); os.IsNotExist(err) {
		m.claims = nil
		m.status = StatusUnlicensed
		m.lastErr = ErrLicenseNotFound
		logs.Warnf("license: %s not found, system is unlicensed", m.licFilePath)
		return nil
	}

	data, err := os.ReadFile(m.licFilePath)
	if err != nil {
		m.status = StatusUnlicensed
		m.lastErr = err
		logs.Errorf("license: read file %s error: %v", m.licFilePath, err)
		return err
	}

	var env LicenseEnvelope
	if err := json.Unmarshal(data, &env); err != nil {
		m.status = StatusInvalidSignature
		m.lastErr = err
		logs.Errorf("license: unmarshal %s error: %v", m.licFilePath, err)
		return err
	}

	claims, err := VerifyEnvelope(&env, m.publicKey)
	if err != nil {
		m.status = StatusInvalidSignature
		m.lastErr = err
		logs.Errorf("license: verification failed for %s: %v", m.licFilePath, err)
		return err
	}

	m.claims = claims
	m.updateStatusLocked()

	logs.Infof("license: loaded valid license for %s (status: %s, expires: %s, maxDevices: %d)",
		claims.CustomerName, m.status, claims.FormatExpiresAt(), claims.MaxDevices)
	return nil
}

func (m *Manager) updateStatusLocked() {
	if m.claims == nil {
		m.status = StatusUnlicensed
		return
	}

	now := time.Now()
	// 1. 机器指纹检查
	if len(m.claims.Fingerprint) > 0 {
		targetFp := NormalizeFingerprint(m.claims.Fingerprint)
		currFp := NormalizeFingerprint(m.cachedFp)
		if targetFp != currFp {
			m.status = StatusFingerprintMismatch
			m.lastErr = ErrLicenseFingerprintMismatch
			return
		}
	}

	// 2. 时间检查
	if m.claims.ExpiresAt > 0 && now.Unix() > m.claims.ExpiresAt {
		m.status = StatusExpired
		m.lastErr = ErrLicenseExpired
		return
	}

	if m.claims.ExpiresAt > 0 && now.Unix() >= m.claims.ExpiresAt-15*86400 {
		m.status = StatusExpiringSoon
		m.lastErr = nil
		return
	}

	m.status = StatusActive
	m.lastErr = nil
}

// CheckValid 检查当前 License 是否允许正常业务运行
func (m *Manager) CheckValid() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	switch m.status {
	case StatusActive, StatusExpiringSoon:
		return nil
	case StatusUnlicensed:
		return errors.New("系统未授权，请联系管理员上传有效 License 证书")
	case StatusExpired:
		return fmt.Errorf("系统 License 授权已过期 (%s)，请续期授权", m.claims.FormatExpiresAt())
	case StatusFingerprintMismatch:
		return errors.New("系统硬件指纹与 License 授权不匹配")
	case StatusInvalidSignature:
		return errors.New("License 签名无效或已被篡改")
	default:
		return errors.New("License 处于无效状态")
	}
}

// CheckCanAddDevice 检查是否允许新增设备（配额检查）
func (m *Manager) CheckCanAddDevice(increment int) error {
	if err := m.CheckValid(); err != nil {
		return err
	}

	m.mu.RLock()
	claims := m.claims
	m.mu.RUnlock()

	if claims == nil || claims.MaxDevices <= 0 {
		return nil // 0 表示无限制
	}

	currentCount, err := countTotalDevices()
	if err != nil {
		logs.Errorf("license: count total devices failed: %v", err)
		return nil // 容错，避免统计查询异常阻塞业务
	}

	if currentCount+int64(increment) > claims.MaxDevices {
		return fmt.Errorf("已达到 License 最大允许接入设备数上限 (当前: %d 台, 上限: %d 台)，无法新增", currentCount, claims.MaxDevices)
	}

	return nil
}

// IsRequireRedirect 是否需要重定向至授权页面/受限状态（纯内存状态判断，极速无额外开销）
func (m *Manager) IsRequireRedirect() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status == StatusUnlicensed || m.status == StatusExpired || m.status == StatusFingerprintMismatch || m.status == StatusInvalidSignature
}

// GetInfo 获取当前系统授权详情
func (m *Manager) GetInfo() *LicenseInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	currentDevices, _ := countTotalDevices()

	info := &LicenseInfo{
		Status:             m.status,
		StatusText:         getStatusText(m.status),
		MachineFingerprint: m.cachedFp,
		CurrentDevices:     currentDevices,
		IsLicensed:         m.status == StatusActive || m.status == StatusExpiringSoon,
		RequireRedirect:    m.status == StatusUnlicensed || m.status == StatusExpired || m.status == StatusFingerprintMismatch || m.status == StatusInvalidSignature,
	}

	if m.claims != nil {
		info.LicenseId = m.claims.LicenseId
		info.CustomerName = m.claims.CustomerName
		info.IssuedAt = m.claims.IssuedAt
		info.IssuedAtFormatted = m.claims.FormatIssuedAt()
		info.ExpiresAt = m.claims.ExpiresAt
		info.ExpiresAtFormatted = m.claims.FormatExpiresAt()
		info.MaxDevices = m.claims.MaxDevices
		info.Fingerprint = m.claims.Fingerprint
	}

	return info
}

func countTotalDevices() (count int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			count = 0
			err = fmt.Errorf("count devices recover: %v", r)
		}
	}()
	o := orm.NewOrm()
	qs := o.QueryTable(models.Device{})
	return qs.Count()
}

func getStatusText(status string) string {
	switch status {
	case StatusActive:
		return "正常生效中"
	case StatusExpiringSoon:
		return "即将到期 (请及时续期)"
	case StatusExpired:
		return "已过期"
	case StatusFingerprintMismatch:
		return "机器指纹不匹配"
	case StatusInvalidSignature:
		return "签名无效/被篡改"
	default:
		return "未授权"
	}
}
