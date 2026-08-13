// Package app is the composition root: construct long-lived dependencies and
// own process Start/Stop order. Phase A/B keep package-level Reg* for compatibility.
package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-iot/pkg/api/eventpush"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core"
	"go-iot/pkg/es"
	"go-iot/pkg/logger"
	"go-iot/pkg/models"
	"go-iot/pkg/models/base"
	modelNetWork "go-iot/pkg/models/network"
	"go-iot/pkg/network/platform/mqtt5"
	"go-iot/pkg/option"
	"go-iot/pkg/redis"
	"go-iot/pkg/ruleengine"
	"go-iot/pkg/store"
	"go-iot/pkg/timeseries"
)

// App holds process-scoped dependencies. Global Reg* still used by existing code;
// new code should prefer fields on App.
type App struct {
	Opt *option.Options

	// Devices is registered into core for compatibility; also kept on App for visibility.
	Devices core.DeviceStore
	// Sessions 设备连接会话表（内存实现，可替换）。
	Sessions core.SessionManager
	// OfflineCmds 离线功能调用队列（生产环境 Redis 实现）。
	OfflineCmds core.OfflineCommandQueue
	// NodeInvoker 跨节点同步调用（HTTP RPC）；功能调用经 DeviceGateway 使用。
	NodeInvoker core.NodeInvoker

	API *web.Server

	// logReloadCancel 停止日志级别热刷新协程。
	logReloadCancel context.CancelFunc
	// retentionCancel 停止 ES 时序保留清理协程。
	retentionCancel context.CancelFunc
}

// New constructs dependencies and fills legacy package globals (Reg*/Config).
// Does not start network or HTTP listeners.
func New(opt *option.Options) (*App, error) {
	if opt == nil {
		return nil, fmt.Errorf("options is nil")
	}

	// Order matters for legacy globals:
	// cluster before netclient restore (Shard uses cluster.Enabled);
	// es/redis before store and model registration.
	cluster.Config(opt)
	es.Config(opt)
	timeseries.ConfigTdengine(opt)
	redis.Config(opt)
	ruleengine.Config(opt)

	devices := store.NewRedisStore()
	sessions := core.NewMemorySessionManager()
	offlineQueue := store.NewRedisOfflineCommandQueue()
	nodeInvoker := cluster.NewHTTPNodeInvoker()

	core.RegDeviceStore(devices)
	core.RegSessionManager(sessions)
	core.RegOfflineCommandQueue(offlineQueue)
	core.RegNodeInvoker(nodeInvoker)

	a := &App{
		Opt:         opt,
		Devices:     devices,
		Sessions:    sessions,
		OfflineCmds: offlineQueue,
		NodeInvoker: nodeInvoker,
		API:         web.NewServer(opt.APIAddr),
	}
	logger.Infof("app assembled: api=%s devices=%s sessions=memory offline_cmds=redis cluster=%v node_invoker=http",
		opt.APIAddr, devices.Id(), cluster.Enabled())
	return a, nil
}

// Start runs bootstrap in a fixed order.
//
//  1. Register ES models
//  2. Seed defaults (admin user, empty network ports)
//  3. restoreRuntime: menu + network/rule/notify/client recovery
//  4. eventpush (WebSocket device event fan-out)
//  5. Built-in MQTT5 broker process
//  6. HTTP API (non-blocking)
//
// Requires main blank-import registry so api.Resources and protocol factories exist
// before restoreRuntime / API start.
func (a *App) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	logger.Infof("app start begin")

	models.RegisterModels()
	logger.Infof("app start: models registered")

	// 默认数据（admin 初始密码来自配置 admin.password / GOIOT_ADMIN_PASSWORD，bcrypt 入库）
	base.EnsureDefaultAdmin(a.Opt.AdminPassword())
	modelNetWork.EnsureDefaultNetworks()
	logger.Infof("app start: seed data ensured (admin, default networks)")

	// 菜单 + 运行态恢复
	a.restoreRuntime()

	// 管理端设备事件 WebSocket 推送
	eventpush.Start()

	if a.NodeInvoker != nil && a.NodeInvoker.Enabled() {
		logger.Infof("app start: cluster node invoker enabled (localId=%s, rpc=HTTP %s)",
			a.NodeInvoker.LocalID(), cluster.CmdInvokePath)
	} else {
		logger.Infof("app start: cluster disabled (single node)")
	}

	if err := mqtt5.Start(); err != nil {
		return fmt.Errorf("mqtt5 server start: %w", err)
	}
	logger.Infof("app start: goiot mqtt5 started")

	if err := a.API.Start(); err != nil {
		return fmt.Errorf("api server start: %w", err)
	}
	logger.Infof("app start: api server started (non-blocking)")

	a.startLogLevelReloader()
	a.startEsRetentionJob()

	logger.Infof("app start end")
	return nil
}

// Stop shuts down in reverse order. Product network servers are not tracked yet (Phase C).
func (a *App) Stop(ctx context.Context) error {
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
	}
	logger.Infof("app stop begin")

	if a.logReloadCancel != nil {
		a.logReloadCancel()
		a.logReloadCancel = nil
	}
	if a.retentionCancel != nil {
		a.retentionCancel()
		a.retentionCancel = nil
	}

	var firstErr error
	if a.API != nil {
		if err := a.API.Shutdown(ctx); err != nil {
			logger.Errorf("api shutdown: %v", err)
			firstErr = err
		}
	}

	if err := mqtt5.Stop(); err != nil {
		logger.Errorf("mqtt5 stop: %v", err)
		if firstErr == nil {
			firstErr = err
		}
	}

	logger.Infof("app stop end")
	return firstErr
}

// Run starts the app and blocks until SIGINT/SIGTERM, then Stop.
func (a *App) Run() error {
	if err := a.Start(context.Background()); err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	logger.Infof("received signal %v, shutting down", sig)

	stopCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return a.Stop(stopCtx)
}

// startEsRetentionJob 启动 ES 时序过期月索引清理。
// retention-check-hours<=0 不启动；集群模式下仅 index==0 执行。
func (a *App) startEsRetentionJob() {
	if a == nil {
		return
	}
	if cluster.Enabled() {
		local := cluster.LocalNode()
		if local.Index != 0 {
			logger.Infof("es retention job skipped on cluster node index=%d (only index=0 runs)", local.Index)
			return
		}
	}
	// 以 es.Config 写入的 DefaultEsConfig 为准（New 时已 Config）
	months := es.DefaultEsConfig.RetentionMonths
	hours := es.DefaultEsConfig.RetentionCheckHours
	if hours <= 0 {
		logger.Infof("es retention job disabled (es.retention-check-hours=%d)", hours)
		return
	}
	a.retentionCancel = timeseries.StartEsRetentionLoop(months, hours, nil)
}

// startLogLevelReloader 定时从配置文件刷新 logs.level。
// ReloadInterval<=0 或未配置 config-file 时不启动。
func (a *App) startLogLevelReloader() {
	if a == nil || a.Opt == nil {
		return
	}
	intervalSec := a.Opt.Log.ReloadInterval
	if intervalSec <= 0 {
		logger.Infof("log level hot-reload disabled (logs.reload-interval=%d)", intervalSec)
		return
	}
	if a.Opt.ConfigFile == "" {
		logger.Infof("log level hot-reload skipped: config-file empty")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.logReloadCancel = cancel
	interval := time.Duration(intervalSec) * time.Second
	configFile := a.Opt.ConfigFile

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		logger.Infof("log level hot-reload started: file=%s interval=%s current=%s",
			configFile, interval, logger.GetLevel())
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				level, err := a.Opt.ReadLogLevelFromFile()
				if err != nil {
					logger.Warnf("reload log level from %s failed: %v", configFile, err)
					continue
				}
				// 规范化后再比较，避免大小写/空格导致重复设置
				normalized := logger.ParseLevel(level).String()
				current := logger.GetLevel()
				if normalized == current {
					continue
				}
				logger.SetLevel(normalized)
				a.Opt.Log.Level = normalized
				logger.Infof("log level reloaded from config: %s -> %s", current, normalized)
			}
		}
	}()
}
