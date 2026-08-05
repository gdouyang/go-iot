package cluster

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go-iot/pkg/common"
	"go-iot/pkg/core"

	logs "go-iot/pkg/logger"
)

// CmdInvokePath 集群内部功能调用 API（请求-响应）。
const CmdInvokePath = "/api/cluster/cmd-invoke"

// HTTPNodeInvoker 通过 HTTP + 集群 token 同步调用对端节点。
type HTTPNodeInvoker struct {
	// 构造时快照；Enabled/LocalID 仍读 registry，便于测试替换 registry 后一致。
}

// NewHTTPNodeInvoker 在 Config 之后构造。
func NewHTTPNodeInvoker() *HTTPNodeInvoker {
	return &HTTPNodeInvoker{}
}

func (h *HTTPNodeInvoker) Enabled() bool   { return Enabled() }
func (h *HTTPNodeInvoker) LocalID() string { return GetClusterId() }

func (h *HTTPNodeInvoker) InvokeRemote(ctx context.Context, nodeID string, msg core.FuncInvoke, timeout time.Duration) *common.Err {
	if !Enabled() {
		return common.NewErr500("cluster not enable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	node := FindNode(nodeID)
	if node == nil {
		return common.NewErr500(fmt.Sprintf("cluster node not found: %s", nodeID))
	}
	if len(node.Url) == 0 {
		return common.NewErr500(fmt.Sprintf("cluster node has no url: %s", nodeID))
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return common.NewErr500(err.Error())
	}

	urlStr := node.Url + CmdInvokePath
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, bytes.NewReader(body))
	if err != nil {
		return common.NewErr500(err.Error())
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set(X_Cluster_Request, Token())
	req.Header.Set(X_Cluster_Timeout, fmt.Sprintf("%d", int(timeout.Seconds())))

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		logs.Warnf("cluster invoke remote %s: %v", nodeID, err)
		return common.NewErr500(fmt.Sprintf("cluster invoke failed: %v", err))
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return common.NewErr500(err.Error())
	}
	var r common.JsonResp
	if err := json.Unmarshal(b, &r); err != nil {
		return common.NewErr500(fmt.Sprintf("cluster invoke bad response: %v", err))
	}
	if r.Success {
		return nil
	}
	code := r.Code
	if code == 0 {
		code = http.StatusBadRequest
	}
	msgText := r.Msg
	if len(msgText) == 0 {
		msgText = "cluster invoke failed"
	}
	return common.NewErr(code, msgText)
}

var _ core.NodeInvoker = (*HTTPNodeInvoker)(nil)
