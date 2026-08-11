package codec

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"go-iot/pkg/option"
)

// checkScriptHTTPAllowed 校验脚本 HttpRequest 是否允许访问 targetURL。
// 关闭 script.http-enabled 或命中私网（当 block-private）时返回错误。
func checkScriptHTTPAllowed(rawURL string) error {
	if !option.ScriptHTTPEnabled() {
		return fmt.Errorf("script HTTP is disabled (script.http-enabled=false)")
	}
	if !option.ScriptHTTPBlockPrivate() {
		return nil
	}
	return rejectPrivateOrLocalURL(rawURL)
}

// rejectPrivateOrLocalURL 禁止 loopback / 链路本地 / 私网 / 未指定地址（SSRF 基线）。
func rejectPrivateOrLocalURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("url host is empty")
	}
	// 字面量 localhost
	if strings.EqualFold(host, "localhost") {
		return fmt.Errorf("script HTTP blocks private/local address: %s", host)
	}
	ip := net.ParseIP(host)
	if ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("script HTTP blocks private/local address: %s", host)
		}
		return nil
	}
	// 主机名：解析 A/AAAA，任一条私网则拒绝
	addrs, err := net.LookupIP(host)
	if err != nil {
		// 解析失败时拒绝，避免绕过
		return fmt.Errorf("script HTTP resolve host %q: %w", host, err)
	}
	if len(addrs) == 0 {
		return fmt.Errorf("script HTTP resolve host %q: no addresses", host)
	}
	for _, a := range addrs {
		if isBlockedIP(a) {
			return fmt.Errorf("script HTTP blocks private/local address: %s -> %s", host, a.String())
		}
	}
	return nil
}

func isBlockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	// 额外：IPv4 文档/基准测试等（IsPrivate 已覆盖 10/8, 172.16/12, 192.168/16）
	// 169.254.0.0/16 已由 IsLinkLocalUnicast 覆盖
	return false
}
