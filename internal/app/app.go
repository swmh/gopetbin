package app

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/swmh/gopetbin/internal/server"
	"github.com/swmh/gopetbin/internal/service"
)

type Config struct {
	Storage           service.Storage
	Cache             service.Cache
	FileCache         service.FileCache
	Repo              service.Repository
	Locker            service.Locker
	Logger            *slog.Logger
	Addr              string
	PublicPath        string
	DefaultExpiration time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IDLength          int
	MaxSize           int64
	MaxFileMemory     int64
}

type App struct {
	server *server.Server
}

func New(c Config) (*App, error) {
	serviceConfig := service.Config{
		Storage:       c.Storage,
		Repo:          c.Repo,
		Cache:         c.Cache,
		FileCache:     c.FileCache,
		Locker:        c.Locker,
		Logger:        c.Logger,
		IDLength:      c.IDLength,
		DefaultExpire: c.DefaultExpiration,
	}

	srvc, err := service.New(serviceConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize service: %w", err)
	}

	serverConfig := server.Config{
		Service:       srvc,
		Logger:        c.Logger,
		Addr:          c.Addr,
		ReadTimeout:   c.ReadTimeout,
		WriteTimout:   c.WriteTimeout,
		MaxSize:       c.MaxSize,
		MaxFileMemory: c.MaxFileMemory,
		PublicPath:    c.PublicPath,
	}

	return &App{
		server.New(serverConfig),
	}, nil
}

func (a *App) Run() error {
	return fmt.Errorf("server running error: %w", a.server.Run())
}

func (a *App) Shutdown(ctx context.Context) error {
	return fmt.Errorf("server shutdown error: %w", a.server.Shutdown(ctx))
}
