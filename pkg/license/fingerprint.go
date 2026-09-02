package license

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

// HardwareInfo 机器硬件特征
type HardwareInfo struct {
	OS          string   `json:"os"`
	Arch        string   `json:"arch"`
	Hostname    string   `json:"hostname"`
	SystemUUID  string   `json:"systemUuid"`
	MacList     []string `json:"macList"`
	Fingerprint string   `json:"fingerprint"`
}

// GetMachineFingerprint 获取当前服务器的机器指纹码
func GetMachineFingerprint() (string, error) {
	info, err := GetHardwareInfo()
	if err != nil {
		return "", err
	}
	return info.Fingerprint, nil
}

// GetHardwareInfo 提取机器的硬件信息并生成指纹
func GetHardwareInfo() (*HardwareInfo, error) {
	hostname, _ := os.Hostname()
	macs := getValidMacAddresses()
	uuid := getSystemUUID()

	var sb strings.Builder
	sb.WriteString("SYS_UUID:" + strings.TrimSpace(uuid) + ";")
	sb.WriteString("MACS:" + strings.Join(macs, ",") + ";")
	sb.WriteString("ARCH:" + runtime.GOARCH + ";")

	rawDigest := sha256.Sum256([]byte(sb.String()))
	hexStr := strings.ToUpper(hex.EncodeToString(rawDigest[:]))

	fingerprint := formatFingerprint(hexStr[:32])

	return &HardwareInfo{
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		Hostname:    hostname,
		SystemUUID:  uuid,
		MacList:     macs,
		Fingerprint: fingerprint,
	}, nil
}

func formatFingerprint(s string) string {
	var parts []string
	runes := []rune(s)
	for i := 0; i < len(runes); i += 4 {
		end := i + 4
		if end > len(runes) {
			end = len(runes)
		}
		parts = append(parts, string(runes[i:end]))
	}
	return strings.Join(parts, "-")
}

func getValidMacAddresses() []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var macs []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagPointToPoint != 0 {
			continue
		}
		mac := iface.HardwareAddr.String()
		if len(mac) == 0 {
			continue
		}
		if strings.Trim(strings.ReplaceAll(mac, ":", ""), "0") == "" {
			continue
		}
		macs = append(macs, strings.ToLower(mac))
	}
	sort.Strings(macs)
	return macs
}

func getSystemUUID() string {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("powershell", "-NoProfile", "-Command", "Get-CimInstance -ClassName Win32_ComputerSystemProduct | Select-Object -ExpandProperty UUID")
		out, err := cmd.Output()
		if err == nil && len(strings.TrimSpace(string(out))) > 0 {
			return strings.TrimSpace(string(out))
		}
		cmd2 := exec.Command("wmic", "csproduct", "get", "uuid")
		out2, err2 := cmd2.Output()
		if err2 == nil {
			lines := strings.Split(string(out2), "\n")
			for _, line := range lines {
				trimmed := strings.TrimSpace(line)
				if len(trimmed) > 0 && !strings.EqualFold(trimmed, "uuid") {
					return trimmed
				}
			}
		}
	case "linux":
		if data, err := os.ReadFile("/sys/class/dmi/id/product_uuid"); err == nil {
			trimmed := strings.TrimSpace(string(data))
			if len(trimmed) > 0 {
				return trimmed
			}
		}
		if data, err := os.ReadFile("/etc/machine-id"); err == nil {
			trimmed := strings.TrimSpace(string(data))
			if len(trimmed) > 0 {
				return trimmed
			}
		}
	case "darwin":
		cmd := exec.Command("ioreg", "-rd1", "-c", "IOPlatformExpertDevice")
		out, err := cmd.Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				if strings.Contains(line, "IOPlatformUUID") {
					parts := strings.Split(line, "=")
					if len(parts) == 2 {
						return strings.Trim(strings.TrimSpace(parts[1]), `"`)
					}
				}
			}
		}
	}
	h, _ := os.Hostname()
	return fmt.Sprintf("HOST-%s", h)
}

func NormalizeFingerprint(fp string) string {
	cleaned := strings.ReplaceAll(fp, "-", "")
	cleaned = strings.ReplaceAll(cleaned, ":", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	return strings.ToLower(strings.TrimSpace(cleaned))
}
