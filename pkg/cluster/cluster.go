package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go-iot/pkg/common"
	"go-iot/pkg/option"

	logs "go-iot/pkg/logger"
)

const (
	X_Cluster_Request = "x-cluster-request"
	X_Cluster_Timeout = "x-cluster-timeout"
)

// ClusterNode 集群节点。
type ClusterNode struct {
	Name     string    `json:"name"`
	Url      string    `json:"url"`
	Index    int       `json:"index"`
	Alive    bool      `json:"alive"`
	LastSeen time.Time `json:"lastSeen,omitempty"`
}

func GetClusterId() string {
	return defaultRegistry.LocalID()
}

// LocalNode 返回本节点信息快照。
func LocalNode() ClusterNode {
	return defaultRegistry.Local()
}

// Enabled 集群是否开启。
func Enabled() bool {
	return defaultRegistry.Enabled()
}

// Size 节点总数（本机+对端配置数）。
func Size() int {
	return defaultRegistry.Size()
}

// Token 集群共享凭证。
func Token() string {
	return defaultRegistry.Token()
}

// FindNode 按节点 name 查找（含本节点）。
func FindNode(name string) *ClusterNode {
	return defaultRegistry.FindByName(name)
}

// ListNodes 返回本机+对端快照（运维/API）。
func ListNodes() []ClusterNode {
	return defaultRegistry.ListAll()
}

// OwnerIndex 计算 key 的分片序号（crc32 % Size）。
// 节点 Index 须在配置中互不重复且落在 [0, Size)。
func OwnerIndex(key string) int {
	size := Size()
	if size <= 0 {
		return 0
	}
	v := crc32.ChecksumIEEE([]byte(key))
	return int(v % uint32(size))
}

// Shard 本节点是否为 key 的分片负责节点（出站 client 等无会话场景）。
func Shard(str string) bool {
	return OwnerIndex(str) == defaultRegistry.LocalIndex()
}

// FindNodeByIndex 按 Index 查找节点（含本机）。对端 Index 在 keepalive 后可用。
func FindNodeByIndex(index int) *ClusterNode {
	local := LocalNode()
	if local.Index == index {
		cp := local
		cp.Alive = true
		return &cp
	}
	for _, n := range defaultRegistry.ListPeers() {
		if n.Index == index {
			cp := n
			return &cp
		}
	}
	return nil
}

// ResolveShardOwner 解析分片负责节点。
// isLocal：本机执行；remote 非 nil 时转发到该节点（需已有 Name）。
func ResolveShardOwner(key string) (isLocal bool, remote *ClusterNode, err error) {
	if !Enabled() {
		return true, nil, nil
	}
	idx := OwnerIndex(key)
	if idx == defaultRegistry.LocalIndex() {
		return true, nil, nil
	}
	n := FindNodeByIndex(idx)
	if n == nil {
		return false, nil, fmt.Errorf("shard owner index=%d not found for key=%s (check peer index/keepalive)", idx, key)
	}
	if len(n.Name) == 0 {
		return false, nil, fmt.Errorf("shard owner index=%d has no name yet (wait keepalive) url=%s", idx, n.Url)
	}
	if !n.Alive && n.Name != GetClusterId() {
		// 仍允许尝试转发；调用方 SingleInvoke 会报错
		logs.Warnf("shard owner %s index=%d may be offline", n.Name, idx)
	}
	return false, n, nil
}

// Config 配置集群并启动 keepalive 循环。
func Config(opt *option.Options) {
	local := ClusterNode{
		Name:  opt.Cluster.Name,
		Url:   opt.Cluster.Url,
		Index: opt.Cluster.Index,
		Alive: true,
	}
	var peerURLs []string
	for _, u := range strings.Split(opt.Cluster.Hosts, ",") {
		u = strings.TrimSpace(u)
		if len(u) > 0 {
			peerURLs = append(peerURLs, u)
		}
	}
	defaultRegistry.configure(local, peerURLs, opt.Cluster.Token, opt.Cluster.Enabled)

	if opt.Cluster.Enabled {
		logs.Infof("cluster enabled: name=%s index=%d peers=%d", local.Name, local.Index, len(peerURLs))
		go keepaliveLoop()
	}
}

func keepaliveLoop() {
	for {
		time.Sleep(5 * time.Second)
		for _, snap := range defaultRegistry.ListPeers() {
			peer, ok := probeKeepalive(snap.Url)
			if !ok {
				defaultRegistry.MarkPeer(snap.Url, false, nil)
				if snap.Alive {
					logs.Warnf("cluster offline url=%s name=%s index=%v", snap.Url, snap.Name, snap.Index)
				}
				continue
			}
			defaultRegistry.MarkPeer(snap.Url, true, peer)
			if !snap.Alive {
				logs.Infof("cluster online url=%s name=%s index=%v", snap.Url, peer.Name, peer.Index)
			}
		}
	}
}

// SingleInvoke 将 HTTP 请求转发到指定 name 的节点（原样克隆，带集群 token）。
func SingleInvoke(clusterId string, req *http.Request) (*common.JsonResp, error) {
	if !Enabled() {
		return nil, errors.New("cluster not enable")
	}
	n := FindNode(clusterId)
	if n == nil {
		return nil, fmt.Errorf("cluster node not found: %s", clusterId)
	}
	if n.Name == GetClusterId() {
		return nil, errors.New("cannot single-invoke local node")
	}
	return invokeNode(n, req)
}

// BroadcastInvoke 广播到所有存活对端（产品/规则发布等）。
func BroadcastInvoke(req *http.Request) error {
	if !Enabled() {
		return nil
	}
	for _, n := range defaultRegistry.ListAlivePeers() {
		if _, err := invokeNode(&n, req); err != nil {
			return err
		}
	}
	return nil
}

// Keepalive 处理对端上报的节点信息。
func Keepalive(c ClusterNode) {
	defaultRegistry.UpsertFromKeepalive(c)
}

func invokeNode(n *ClusterNode, req *http.Request) (*common.JsonResp, error) {
	if n == nil || len(n.Url) == 0 {
		return nil, errors.New("invalid cluster node")
	}
	req2 := req.Clone(context.Background())
	req2.Header.Set(X_Cluster_Request, Token())
	u, err := url.ParseRequestURI(n.Url + req2.RequestURI)
	if err != nil {
		return nil, err
	}
	req2.URL = u
	req2.RequestURI = ""

	timeout := 10
	if s := req.Header.Get(X_Cluster_Timeout); len(s) > 0 {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			timeout = v
		}
	}

	client := http.Client{Timeout: time.Second * time.Duration(timeout)}
	resp, err := client.Do(req2)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var r common.JsonResp
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("cluster response decode: %w", err)
	}
	return &r, nil
}

func probeKeepalive(peerURL string) (*ClusterNode, bool) {
	client := http.Client{Timeout: 3 * time.Second}
	uri, err := url.ParseRequestURI(peerURL + "/api/cluster/keepalive")
	if err != nil {
		logs.Errorf("keepalive url error: %v", err)
		return nil, false
	}
	local := LocalNode()
	body, _ := json.Marshal(local)
	req, err := http.NewRequest(http.MethodPost, uri.String(), bytes.NewReader(body))
	if err != nil {
		return nil, false
	}
	req.Header.Set(X_Cluster_Request, Token())
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := client.Do(req)
	if err != nil {
		logs.Errorf("keepalive error: %v", err)
		return nil, false
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		logs.Errorf("keepalive status=%d body=%s", resp.StatusCode, string(raw))
		return nil, false
	}
	var jr common.JsonResp
	if err := json.Unmarshal(raw, &jr); err != nil || jr.Result == nil {
		// 旧节点可能只返回 success 无 result
		return &ClusterNode{Url: peerURL}, true
	}
	b, _ := json.Marshal(jr.Result)
	var peer ClusterNode
	if err := json.Unmarshal(b, &peer); err != nil {
		return &ClusterNode{Url: peerURL}, true
	}
	if len(peer.Url) == 0 {
		peer.Url = peerURL
	}
	return &peer, true
}
