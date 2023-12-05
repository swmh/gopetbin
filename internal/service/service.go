package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"time"

	l "github.com/swmh/gopetbin/internal/logger"
	"github.com/swmh/gopetbin/internal/server"
)

type Paste struct {
	Name       string
	Expire     time.Time
	BurnAfter  int
	IsBurnable bool
}

type ToReadCloser struct {
	io.Reader
}

func (t ToReadCloser) Close() error {
	return nil
}

var errNoSuchPaste = errors.New("no such paste")

type NoSuchPasteChecker interface {
	IsNoSuchPaste(err error) bool
}

type Cache interface {
	Set(ctx context.Context, key string, value Paste) error
	SetError(ctx context.Context, key string, ttl time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	IsError(ctx context.Context, value string) bool
	Unmarshal(ctx context.Context, value string) (Paste, error)
	NoSuchPasteChecker
}

type FileCache interface {
	Set(ctx context.Context, key string, value []byte) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	NoSuchPasteChecker
}

type Storage interface {
	PutFile(ctx context.Context, name string, data io.Reader, size int64) error
	GetFile(ctx context.Context, name string) (io.ReadCloser, error)
	IsPasteExist(ctx context.Context, name string) bool
	NoSuchPasteChecker
}

type Repository interface {
	CreatePaste(ctx context.Context, id string, name string, expire time.Time, burn int) error
	GetPaste(ctx context.Context, id string) (Paste, error)
	Readed(ctx context.Context, id string) error
	NoSuchPasteChecker
}

type Mutex interface {
	Unlock(ctx context.Context) error
}

type Locker interface {
	Lock(ctx context.Context, id string) (Mutex, error)
}

type Config struct {
	Storage   Storage
	Repo      Repository
	Cache     Cache
	FileCache FileCache
	Locker    Locker
	Logger    *slog.Logger

	IDLength      int
	DefaultExpire time.Duration
}

type Service struct {
	storage   Storage
	repo      Repository
	cache     Cache
	fileCache FileCache
	locker    Locker
	logger    *slog.Logger

	idLength      int
	defaultExpire time.Duration
}

func New(c Config) (*Service, error) {
	if c.IDLength <= 0 {
		return nil, errors.New("length must be >= 0")
	}

	if c.DefaultExpire <= 0 {
		return nil, errors.New("default expire must be >= 0")
	}

	return &Service{
		storage:       c.Storage,
		repo:          c.Repo,
		cache:         c.Cache,
		fileCache:     c.FileCache,
		locker:        c.Locker,
		logger:        c.Logger,
		idLength:      c.IDLength,
		defaultExpire: c.DefaultExpire,
	}, nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func (s *Service) getID() string {
	b := make([]rune, s.idLength)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}

	return string(b)
}

func getName(data []byte) string {
	h := md5.Sum(data)
	return hex.EncodeToString(h[:])
}

func (s *Service) IsNoSuchPaste(err error) bool {
	return s.repo.IsNoSuchPaste(err) ||
		s.storage.IsNoSuchPaste(err) ||
		s.cache.IsNoSuchPaste(err) ||
		s.fileCache.IsNoSuchPaste(err) ||
		errors.Is(err, errNoSuchPaste)
}

func (s *Service) getPaste(ctx context.Context, id string) (Paste, error) {
	mutex, err := s.locker.Lock(ctx, id)
	if err != nil {
		return Paste{}, fmt.Errorf("cannot acquire lock: %w", err)
	}

	defer func() {
		if err = mutex.Unlock(ctx); err != nil {
			s.logger.Error("Cannot unlock", l.ErrorAttr(err))
		}
	}()

	var paste Paste

	value, err := s.cache.Get(ctx, id)
	if err == nil {
		if s.cache.IsError(ctx, value) {
			return Paste{}, errNoSuchPaste
		}

		paste, err = s.cache.Unmarshal(ctx, value)
	}

	if err != nil {
		if !s.cache.IsNoSuchPaste(err) {
			s.logger.Warn("Cannot get value from cache", slog.String("key", id), l.ErrorAttr(err))
		}

		paste, err = s.repo.GetPaste(ctx, id)
		if err != nil {
			return paste, fmt.Errorf("cannot get paste from repo: %w", err)
		}
	}

	if time.Now().UTC().After(paste.Expire) {
		return paste, fmt.Errorf("paste expired: %w", errNoSuchPaste)
	}

	if paste.IsBurnable {
		if paste.BurnAfter <= 0 {
			return paste, fmt.Errorf("paste already burned: %w", errNoSuchPaste)
		}

		if err = s.repo.Readed(ctx, id); err != nil {
			return paste, fmt.Errorf("cannot read paste in repo: %w", err)
		}

		paste.BurnAfter--
	}

	err = s.cache.Set(ctx, id, paste)
	if err != nil {
		s.logger.Warn("Cannot set value in cache", slog.String("key", id), l.ErrorAttr(err))
	}

	return paste, nil
}

func (s *Service) GetPaste(ctx context.Context, id string) (io.ReadCloser, error) {
	paste, err := s.getPaste(ctx, id)
	if err != nil {
		if s.IsNoSuchPaste(err) {
			if cerr := s.cache.SetError(ctx, id, time.Hour); cerr != nil {
				s.logger.Warn("Cannot set error in cache", slog.String("key", id), l.ErrorAttr(err))
			}
		}

		return nil, err
	}

	var file io.ReadCloser

	file, err = s.fileCache.Get(ctx, id)
	if err != nil {
		if !s.fileCache.IsNoSuchPaste(err) {
			s.logger.Warn("Cannot get value from file cache", slog.String("key", id), l.ErrorAttr(err))
		}

		file, err = s.storage.GetFile(ctx, paste.Name)
		if err != nil {
			return nil, fmt.Errorf("cannot get paste from storage: %w", err)
		}

		data, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("cannot read file: %w", err)
		}

		file = ToReadCloser{bytes.NewReader(data)}

		err = s.fileCache.Set(ctx, id, data)
		if err != nil {
			s.logger.Warn("Cannot set value in file cache", slog.String("key", id), l.ErrorAttr(err))
		}
	}

	return file, nil
}

func (s *Service) CreatePaste(ctx context.Context, paste server.Paste) (string, error) {
	name := getName(paste.Content)
	if !s.storage.IsPasteExist(ctx, name) {
		err := s.storage.PutFile(ctx, name, bytes.NewReader(paste.Content), int64(len(paste.Content)))
		if err != nil {
			return "", err
		}
	}

	t := time.Now().UTC().Add(paste.Expire)
	id := s.getID()

	return id, s.repo.CreatePaste(ctx, id, name, t, paste.BurnAfter)
}
