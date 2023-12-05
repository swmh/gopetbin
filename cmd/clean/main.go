package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/swmh/gopetbin/internal/config"
	"github.com/swmh/gopetbin/internal/db"
	"github.com/swmh/gopetbin/internal/storage"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Config path")
	flag.Parse()

	cfg, err := config.New(configPath)
	if err != nil {
		panic(err)
	}

	c := storage.Config{
		Addr:            cfg.Storage.Addr,
		AccessKeyID:     cfg.Storage.User,
		SecretAccessKey: cfg.Storage.Pass,
		BucketName:      cfg.Storage.Name,
	}

	s, err := storage.New(c)
	if err != nil {
		panic(err)
	}

	db, err := db.New(cfg.DB.Addr, cfg.DB.User, cfg.DB.Pass, cfg.DB.Name)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	expired, err := db.GetExpired(ctx)
	if err != nil {
		panic(err)
	}

	for _, name := range expired {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = s.DeleteFile(ctx, name)
		if err != nil {
			log.Printf("Cannot delete %s: %s\n", name, err)
			continue
		}

		log.Printf("Deleted: %s\n", name)
	}
}
