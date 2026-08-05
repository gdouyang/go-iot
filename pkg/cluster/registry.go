package cluster

import (
	"sync"
	"time"
)

// Registry 集群节点视图：配置引导 + keepalive 回填。
// 并发安全；包级 API 委托到 defaultRegistry。
type Registry struct {
	mu      sync.RWMutex
	enabled bool
	token   string
	local   ClusterNode
	// peers keyed by Url（配置主键）；Name 在 keepalive 后补齐
	peers map[string]*ClusterNode
}

func newRegistry() *Registry {
	return &Registry{
		peers: make(map[string]*ClusterNode),
	}
}

var defaultRegistry = newRegistry()

// GetRegistry 返回进程内节点注册表。
func GetRegistry() *Registry {
	return defaultRegistry
}

func (r *Registry) configure(local ClusterNode, peerURLs []string, token string, enabled bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.enabled = enabled
	r.token = token
	r.local = local
	r.peers = make(map[string]*ClusterNode)
	for _, u := range peerURLs {
		if len(u) == 0 || u == local.Url {
			continue
		}
		r.peers[u] = &ClusterNode{Url: u}
	}
}

func (r *Registry) Enabled() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.enabled
}

func (r *Registry) Token() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.token
}

func (r *Registry) Local() ClusterNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.local
}

func (r *Registry) LocalID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.local.Name
}

func (r *Registry) Size() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.peers) + 1
}

func (r *Registry) LocalIndex() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.local.Index
}

// FindByName 按节点名查找（含本节点）。
func (r *Registry) FindByName(name string) *ClusterNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(name) == 0 {
		return nil
	}
	if r.local.Name == name {
		cp := r.local
		cp.Alive = true
		return &cp
	}
	for _, n := range r.peers {
		if n != nil && n.Name == name {
			cp := *n
			return &cp
		}
	}
	return nil
}

// FindByURL 按 URL 查找对端。
func (r *Registry) FindByURL(u string) *ClusterNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if n, ok := r.peers[u]; ok && n != nil {
		cp := *n
		return &cp
	}
	return nil
}

// ListPeers 返回对端快照（不含本节点）。
func (r *Registry) ListPeers() []ClusterNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ClusterNode, 0, len(r.peers))
	for _, n := range r.peers {
		if n != nil {
			out = append(out, *n)
		}
	}
	return out
}

// ListAll 本节点 + 对端。
func (r *Registry) ListAll() []ClusterNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ClusterNode, 0, len(r.peers)+1)
	local := r.local
	local.Alive = true
	out = append(out, local)
	for _, n := range r.peers {
		if n != nil {
			out = append(out, *n)
		}
	}
	return out
}

// ListAlivePeers 仅存活对端。
func (r *Registry) ListAlivePeers() []ClusterNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ClusterNode, 0, len(r.peers))
	for _, n := range r.peers {
		if n != nil && n.Alive {
			out = append(out, *n)
		}
	}
	return out
}

// UpsertFromKeepalive 对端上报或本机探测成功后更新。
func (r *Registry) UpsertFromKeepalive(c ClusterNode) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(c.Url) == 0 {
		return
	}
	n, ok := r.peers[c.Url]
	if !ok || n == nil {
		n = &ClusterNode{Url: c.Url}
		r.peers[c.Url] = n
	}
	if len(c.Name) > 0 {
		n.Name = c.Name
	}
	n.Index = c.Index
	n.Alive = true
	n.LastSeen = time.Now()
}

// MarkPeer 更新探测结果（按 URL）。
func (r *Registry) MarkPeer(url string, alive bool, peer *ClusterNode) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.peers[url]
	if !ok || n == nil {
		return
	}
	wasAlive := n.Alive
	n.Alive = alive
	if alive {
		n.LastSeen = time.Now()
		if peer != nil {
			if len(peer.Name) > 0 {
				n.Name = peer.Name
			}
			n.Index = peer.Index
		}
	}
	_ = wasAlive
}
