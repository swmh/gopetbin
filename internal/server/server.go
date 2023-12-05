package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

func internalError(w http.ResponseWriter) {
	http.Error(w, "Internal Server Error", http.StatusInternalServerError)
}

type Service interface {
	CreatePaste(ctx context.Context, paste Paste) (string, error)
	GetPaste(ctx context.Context, id string) (io.ReadCloser, error)
	IsNoSuchPaste(error) bool
}

type Config struct {
	Service       Service
	Logger        *slog.Logger
	PublicPath    string
	Addr          string
	MaxSize       int64
	MaxFileMemory int64
	ReadTimeout   time.Duration
	WriteTimout   time.Duration
}

type Server struct {
	service       Service
	logger        *slog.Logger
	server        *http.Server
	publicPath    string
	maxSize       int64
	maxFileMemory int64
}

func requestLogger(l *slog.Logger, method, path string) *slog.Logger {
	return l.With(
		slog.Group(
			"request",
			slog.String("path", path),
			slog.String("method", method),
		),
	)
}

func New(c Config) *Server {
	router := chi.NewRouter()
	server := &http.Server{
		Addr:         c.Addr,
		Handler:      router,
		ReadTimeout:  c.ReadTimeout,
		WriteTimeout: c.WriteTimout,
		ErrorLog:     slog.NewLogLogger(c.Logger.Handler(), slog.LevelInfo),
	}
	api := &Server{
		maxSize:       c.MaxSize,
		service:       c.Service,
		logger:        c.Logger,
		server:        server,
		maxFileMemory: c.MaxFileMemory,
		publicPath:    c.PublicPath,
	}

	log := requestLogger(c.Logger, "POST", "/")
	router.Post("/", api.NewUploadForm(log))

	log = requestLogger(c.Logger, "GET", "/")
	router.Get("/{id}", api.NewGet(log))

	return api
}

func (s *Server) Run() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
