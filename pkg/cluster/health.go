package cluster

// HealthStatus 集群健康等级（语义类似 ES _cluster/health，粒度更粗）。
type HealthStatus string

const (
	// HealthGreen 单机，或集群已启用且配置的对端全部存活。
	HealthGreen HealthStatus = "green"
	// HealthYellow 集群已启用，但部分对端不可达 / 尚无 name（分片或转发可能受影响）。
	HealthYellow HealthStatus = "yellow"
	// HealthRed 集群已启用但本机配置明显不可用（无 name），或全部对端不可达。
	HealthRed HealthStatus = "red"
)

// HealthReport 集群健康快照（本节点视角）。
type HealthReport struct {
	// Status green | yellow | red
	Status HealthStatus `json:"status"`
	// Enabled 是否开启集群模式
	Enabled bool `json:"enabled"`
	// Local 本节点 name
	Local string `json:"local"`
	// LocalURL 本节点 url
	LocalURL string `json:"localUrl"`
	// LocalIndex 本节点分片 index
	LocalIndex int `json:"localIndex"`
	// NumberOfNodes 本机 + 配置对端总数
	NumberOfNodes int `json:"numberOfNodes"`
	// NumberOfNodesAlive 本机(计1) + 存活对端
	NumberOfNodesAlive int `json:"numberOfNodesAlive"`
	// NumberOfPeers 配置的对端数
	NumberOfPeers int `json:"numberOfPeers"`
	// NumberOfPeersAlive 存活对端数
	NumberOfPeersAlive int `json:"numberOfPeersAlive"`
	// UnnamedPeers 尚未回填 name 的对端数（keepalive 未完成）
	UnnamedPeers int `json:"unnamedPeers"`
	// TimedOut 预留（当前同步计算，恒为 false）
	TimedOut bool `json:"timedOut"`
	// Message 人类可读摘要
	Message string `json:"message,omitempty"`
	// Nodes 节点列表（与 ListNodes 一致，便于一次拉取）
	Nodes []ClusterNode `json:"nodes"`
}

// Health 从本节点 Registry 视图生成健康报告。
func Health() HealthReport {
	r := defaultRegistry
	local := r.Local()
	peers := r.ListPeers()
	alivePeers := 0
	unnamed := 0
	for _, p := range peers {
		if p.Alive {
			alivePeers++
		}
		if len(p.Name) == 0 {
			unnamed++
		}
	}

	rep := HealthReport{
		Enabled:            r.Enabled(),
		Local:              local.Name,
		LocalURL:           local.Url,
		LocalIndex:         local.Index,
		NumberOfNodes:      len(peers) + 1,
		NumberOfNodesAlive: alivePeers + 1, // 本机视为存活
		NumberOfPeers:      len(peers),
		NumberOfPeersAlive: alivePeers,
		UnnamedPeers:       unnamed,
		TimedOut:           false,
		Nodes:              r.ListAll(),
	}

	if !rep.Enabled {
		rep.Status = HealthGreen
		rep.Message = "cluster disabled (single node)"
		return rep
	}

	// 集群开启但本机未配置 name：无法正确写 ClusterId / 被转发识别
	if len(local.Name) == 0 {
		rep.Status = HealthRed
		rep.Message = "cluster enabled but local name is empty"
		return rep
	}

	if len(peers) == 0 {
		// 开启集群但无对端：可运行，但失去多节点意义
		rep.Status = HealthYellow
		rep.Message = "cluster enabled but no peers configured"
		return rep
	}

	if alivePeers == 0 {
		rep.Status = HealthRed
		rep.Message = "all peers unreachable"
		return rep
	}

	if alivePeers < len(peers) || unnamed > 0 {
		rep.Status = HealthYellow
		if unnamed > 0 && alivePeers < len(peers) {
			rep.Message = "some peers down or not fully discovered"
		} else if unnamed > 0 {
			rep.Message = "some peers missing name (wait keepalive)"
		} else {
			rep.Message = "some peers unreachable"
		}
		return rep
	}

	rep.Status = HealthGreen
	rep.Message = "all configured peers alive"
	return rep
}
