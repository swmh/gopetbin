package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/swmh/gopetbin/internal/app"
	"github.com/swmh/gopetbin/internal/cache"
	"github.com/swmh/gopetbin/internal/config"
	"github.com/swmh/gopetbin/internal/db"
	"github.com/swmh/gopetbin/internal/lock/redlock"
	"github.com/swmh/gopetbin/internal/storage"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Config path")
	flag.Parse()

	cfg, err := config.New(configPath)
	if err != nil {
		log.Panicf("Cannot load config: %s", err)
	}

	var logLevel slog.Leveler

	switch cfg.App.LogLevel {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel}))

	repo, err := db.New(cfg.DB.Addr, cfg.DB.User, cfg.DB.Pass, cfg.DB.Name)
	if err != nil {
		panic(err)
	}

	storageConfig := storage.Config{
		Addr:            cfg.Storage.Addr,
		AccessKeyID:     cfg.Storage.User,
		SecretAccessKey: cfg.Storage.Pass,
		BucketName:      cfg.Storage.Name,
	}

	strg, err := storage.New(storageConfig)
	if err != nil {
		panic(err)
	}

	cach, err := cache.New(cfg.Cache.Addr, cfg.Cache.User, cfg.Cache.Pass, cfg.Cache.DB)
	if err != nil {
		panic(err)
	}

	fileCache, err := cache.NewFileCache(cfg.FileCache.Addr, cfg.FileCache.User, cfg.FileCache.Pass, cfg.FileCache.DB)
	if err != nil {
		panic(err)
	}

	locker, err := redlock.New(cfg.Locker.Addr, cfg.Locker.User, cfg.Locker.Pass, cfg.Locker.DB)
	if err != nil {
		panic(err)
	}

	c := app.Config{
		Repo:              repo,
		Locker:            locker,
		Storage:           strg,
		Cache:             cach,
		FileCache:         fileCache,
		Logger:            logger,
		Addr:              cfg.App.Addr,
		PublicPath:        cfg.App.PublicPath,
		MaxFileMemory:     cfg.App.MaxFileMemory,
		IDLength:          cfg.App.IDLength,
		MaxSize:           cfg.App.MaxSize,
		DefaultExpiration: time.Duration(cfg.App.DefaultExpiration) * time.Hour,
		ReadTimeout:       time.Duration(cfg.App.ReadTimeout) * time.Second,
		WriteTimeout:      time.Duration(cfg.App.WriteTimeout) * time.Second,
	}

	a, err := app.New(c)
	if err != nil {
		log.Panicf("Initialization failed: %s\n", err)
	}

	go func() {
		err = a.Run()
		logger.Info(fmt.Sprintf("App closed: %s", err))
		os.Exit(0)
	}()

	logger.Info("App started")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	logger.Info(fmt.Sprintf("Get signal: %s", <-sigCh))

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	logger.Info("Stopping app gracefully")
	logger.Info(fmt.Sprintf("App stopped: %s", a.Shutdown(ctx)))
}
