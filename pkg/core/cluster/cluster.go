package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"hash/crc32"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

const (
	X_Cluster_Request = "x-cluster-request"
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
func Config(fn func(key string, call func(string))) {
	fn("cluster.enabled", func(s string) {
		if s == "true" {
			enabled = true
		}
	})
	fn("cluster.name", func(s string) {
		currentNode.Name = s
	})
	fn("cluster.url", func(s string) {
		currentNode.Url = s
	})
	fn("cluster.token", func(s string) {
		token = s
	})
	fn("cluster.index", func(s string) {
		index, err := strconv.Atoi(s)
		if err == nil {
			currentNode.Index = index
		} else {
			logs.Error("cluster.index error:", err)
		}
	})
	fn("cluster.hosts", func(s string) {
		hosts := strings.Split(s, ",")
		for _, url := range hosts {
			if url != currentNode.Url {
				var node ClusterNode
				node.Url = url
				nodes = append(nodes, &node)
			}
		}
	})
	if enabled {
		logs.Info("cluster is enabled")
		go func() {
			for {
				time.Sleep(time.Second * time.Duration(5))
				for _, n := range nodes {
					n.Alive = n.keepalive()
					if !n.Alive {
						logs.Warn("cluster is offline, name:%s, index: %v, url: %s", n.Name, n.Index, n.Url)
					}
				}
			}
		}()
	}
}

func SingleInvoke(cluserId string, req *http.Request) (string, error) {
	if !enabled {
		return "cluster not enable", nil
	}
	for _, n := range nodes {
		if !n.Alive {
			continue
		}
		if n.Name == cluserId {
			resp, err := n.invoke(req)
			if err != nil {
				return resp, err
			}
			break
		}
	}
	return "clusterId not found", nil
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
		if n.Name == c.Name {
			n.Alive = true
		}
	}
}

type ClusterNode struct {
	Name  string `json:"name"`
	Url   string `json:"url"`
	Index int    `json:"index"`
	Alive bool   `json:"-"`
}

func (n *ClusterNode) invoke(req *http.Request) (string, error) {
	if !n.Alive {
		return "", nil
	}
	req2 := req.Clone(context.Background())
	req2.Header.Add(X_Cluster_Request, token)

	client := http.Client{Timeout: time.Second * 3}
	resp, err := client.Do(req2)
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", errors.New(string(b))
	}
	return string(b), nil
}

func (n *ClusterNode) keepalive() bool {
	client := http.Client{Timeout: time.Second * 3}
	uri, err := url.ParseRequestURI(n.Url + "/api/cluster/keepalive")
	if err != nil {
		logs.Error(err)
		return false
	}
	var req *http.Request = &http.Request{
		Method: "post",
		URL:    uri,
		Header: map[string][]string{},
	}
	req.Header.Add(X_Cluster_Request, token)
	req.Header.Add("Content-Type", "application/json; charset=utf-8")
	b, _ := json.Marshal(currentNode)
	req.Body = io.NopCloser(strings.NewReader(string(b)))
	resp, err := client.Do(req)
	if err != nil {
		logs.Error(err)
		return false
	}
	if resp.StatusCode != 200 {
		b, err := io.ReadAll(resp.Body)
		if err != nil {
			logs.Error(err)
			return false
		}
		logs.Error(string(b))
		return false
	}
	return true
}
