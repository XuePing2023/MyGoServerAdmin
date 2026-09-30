// ServerAdmin 后台管理系统启动入口。
package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"serveradmin/internal/api"
	"serveradmin/internal/cache"
	"serveradmin/internal/config"
	"serveradmin/internal/database"
	"serveradmin/internal/pkg/jwtx"
	"serveradmin/internal/pkg/logger"
	"serveradmin/internal/service"
)

func main() {
	cfg := config.Load()

	// 目录准备
	_ = os.MkdirAll(cfg.App.UploadDir, 0o755)
	if dir := filepath.Dir(cfg.Log.File); dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0o755)
	}
	if err := logger.Init(cfg.Log.Level, cfg.Log.File, cfg.Log.MaxSizeMB, cfg.Log.MaxBackups, cfg.Log.MaxAgeDays); err != nil {
		fmt.Println("初始化日志失败:", err)
		os.Exit(1)
	}
	logger.L.Infof("正在启动 %s v%s (env=%s) ...", cfg.App.Name, cfg.App.Version, cfg.App.Env)

	// MongoDB
	mongoTimeout := time.Duration(cfg.Mongo.TimeoutSeconds) * time.Second
	client, db, err := database.Connect(cfg)
	if err != nil {
		logger.L.Fatalf("%v\n提示：请确认 MongoDB 已启动（本地可执行: docker run -d -p 27017:27017 --name serveradmin-mongo mongo:7）", err)
	}
	logger.L.Infof("MongoDB 已连接: %s / 库 %s", cfg.Mongo.URI, cfg.Mongo.Database)

	ctxIdx, cancelIdx := context.WithTimeout(context.Background(), mongoTimeout)
	err = database.EnsureIndexes(ctxIdx, db)
	cancelIdx()
	if err != nil {
		logger.L.Fatalf("初始化索引失败: %v", err)
	}

	// 种子数据
	if cfg.Seed.Enabled {
		ctxSeed, cancelSeed := context.WithTimeout(context.Background(), mongoTimeout)
		if err := service.Seed(ctxSeed, db); err != nil {
			cancelSeed()
			logger.L.Fatalf("初始化种子数据失败: %v", err)
		}
		cancelSeed()
	}

	// JWT
	jwtMgr := jwtx.NewManager(cfg.JWT.Secret, cfg.JWT.Issuer,
		time.Duration(cfg.JWT.AccessExpireMinutes)*time.Minute,
		time.Duration(cfg.JWT.RefreshExpireHours)*time.Hour)

	// Redis 缓存：enabled=false 或连接失败时降级为无缓存模式（直连 MongoDB）
	var cacheClient cache.Cache
	if cfg.Redis.Enabled {
		rdb, err := cache.Connect(context.Background(), &cfg.Redis)
		if err != nil {
			logger.L.Warnf("Redis 连接失败（%s）: %v，缓存已停用", cfg.Redis.Addr, err)
		} else {
			cacheClient = cache.NewRedisCache(rdb)
			defer func() { _ = rdb.Close() }()
			logger.L.Infof("Redis 已连接: %s (db=%d)", cfg.Redis.Addr, cfg.Redis.DB)
		}
	}

	// 服务注册
	reg := service.New(db, cfg, jwtMgr, cacheClient)

	// 令牌黑名单预热 + 定时任务调度器
	bg := context.Background()
	if err := reg.Token.Load(bg); err != nil {
		logger.L.Warnf("加载令牌黑名单失败: %v", err)
	}
	if err := reg.Job.Start(bg); err != nil {
		logger.L.Warnf("启动定时任务调度器失败: %v", err)
	}

	// HTTP 服务
	router := api.NewRouter(cfg, reg, jwtMgr, client)
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.App.Port),
		Handler:      router,
		ReadTimeout:  time.Duration(cfg.App.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.App.WriteTimeout) * time.Second,
	}

	go func() {
		logger.L.Infof("HTTP 服务已启动: http://localhost:%d （内嵌管理界面，默认账号 admin / admin123）", cfg.App.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.L.Fatalf("HTTP 服务异常退出: %v", err)
		}
	}()

	// 优雅停机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.L.Info("收到退出信号，正在优雅停机 ...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.L.Errorf("HTTP 停机异常: %v", err)
	}
	reg.Job.Stop()
	_ = client.Disconnect(shutdownCtx)
	logger.L.Info("服务已退出")
	logger.Sync()
}
