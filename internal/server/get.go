package server

import (
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	l "github.com/swmh/gopetbin/internal/logger"
)

func (s *Server) NewGet(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "id")
		id = strings.TrimSpace(id)
		if id == "" {
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}

		file, err := s.service.GetPaste(r.Context(), id)
		defer func() {
			if file != nil {
				file.Close()
			}
		}()

		if err != nil {
			if s.service.IsNoSuchPaste(err) {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}

			internalError(w)
			s.logger.Error("Cannot get file", l.ErrorAttr(err))

			return
		}

		io.Copy(w, file)
	}
}
