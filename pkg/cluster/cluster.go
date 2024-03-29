package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"go-iot/pkg/common"
	"go-iot/pkg/option"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	logs "go-iot/pkg/logger"
)

const (
	X_Cluster_Request = "x-cluster-request"
	X_Cluster_Timeout = "x-cluster-timeout"
)

var currentNode *ClusterNode = &ClusterNode{}
var nodes []*ClusterNode = []*ClusterNode{}
var enabled bool
var token string

func GetClusterId() string {
	return currentNode.Name
}

// true cluster is enable
func Enabled() bool {
	return enabled
}

// 集群节点数
func Size() int {
	return len(nodes) + 1
}

// 集群token
func Token() string {
	return token
}

// 分片
func Shard(str string) bool {
	v := crc32.ChecksumIEEE([]byte(str))
	mode := v % uint32(Size())
	return mode == uint32(currentNode.Index)
}

// 配置集群
func Config(opt *option.Options) {
	enabled = opt.Cluster.Enable
	currentNode.Name = opt.Cluster.Name
	currentNode.Url = opt.Cluster.Url
	token = opt.Cluster.Token
	currentNode.Index = opt.Cluster.Index
	hosts := strings.Split(opt.Cluster.Hosts, ",")
	for _, url := range hosts {
		if url != currentNode.Url {
			var node ClusterNode
			node.Url = url
			nodes = append(nodes, &node)
		}
	}
	if enabled {
		logs.Infof("cluster is enabled")
		go func() {
			for {
				time.Sleep(time.Second * time.Duration(5))
				for _, n := range nodes {
					alive := n.keepalive()
					if !alive {
						logs.Warnf("cluster is offline url: %s, name: %s, index: %v", n.Url, n.Name, n.Index)
					}
					if !n.Alive && alive {
						logs.Infof("cluster is online url: %s, name: %s, index: %v", n.Url, n.Name, n.Index)
					}
					n.Alive = alive
				}
			}
		}()
	}
}

func SingleInvoke(cluserId string, req *http.Request) (*common.JsonResp, error) {
	if !enabled {
		return nil, errors.New("cluster not enable")
	}
	for _, n := range nodes {
		if !n.Alive {
			continue
		}
		if n.Name == cluserId {
			resp, err := n.invoke(req)
			return resp, err
		}
	}
	return nil, errors.New("clusterId not found")
}

// 广播调用其它节点
func BroadcastInvoke(req *http.Request) error {
	if !enabled {
		return nil
	}
	for _, n := range nodes {
		if !n.Alive {
			continue
		}
		_, err := n.invoke(req)
		if err != nil {
			return err
		}
	}
	return nil
}

func Keepalive(c ClusterNode) {
	for _, n := range nodes {
		if n.Url == c.Url {
			n.Alive = true
			n.Name = c.Name
			n.Index = c.Index
		}
	}
}

type ClusterNode struct {
	Name  string `json:"name"`
	Url   string `json:"url"`
	Index int    `json:"index"`
	Alive bool   `json:"-"`
}

func (n *ClusterNode) invoke(req *http.Request) (*common.JsonResp, error) {
	if !n.Alive {
		return nil, nil
	}
	req2 := req.Clone(context.Background())
	req2.Header.Add(X_Cluster_Request, token)
	u, err := url.ParseRequestURI(n.Url + req2.RequestURI)
	if err != nil {
		return nil, err
	}
	req2.URL = u
	req2.RequestURI = ""

	s_timeout := req.Header.Get("x-cluster-timeout")
	timeout, err := strconv.Atoi(s_timeout)
	if err == nil {
		logs.Warnf("x-cluster-timeout parse error: %v", err)
	}
	if timeout < 1 {
		timeout = 10
	}

	client := http.Client{Timeout: (time.Second * time.Duration(timeout))}
	resp, err := client.Do(req2)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var r common.JsonResp
	json.Unmarshal(b, &r)
	return &r, nil
}

func (n *ClusterNode) keepalive() bool {
	client := http.Client{Timeout: time.Second * 3}
	uri, err := url.ParseRequestURI(n.Url + "/api/cluster/keepalive")
	if err != nil {
		logs.Errorf("keepalive error: %v", err)
		return false
	}
	var req *http.Request = &http.Request{
		Method: "POST",
		URL:    uri,
		Header: map[string][]string{},
	}
	req.Header.Add(X_Cluster_Request, token)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	b, _ := json.Marshal(currentNode)
	req.Body = io.NopCloser(strings.NewReader(string(b)))
	resp, err := client.Do(req)
	if err != nil {
		logs.Errorf("keepalive error: %v", err)
		return false
	}
	if resp.StatusCode != 200 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			logs.Errorf("keepalive error: %v", err)
			return false
		}
		logs.Errorf("keepalive error: %s", string(b))
		return false
	}
	return true
}
