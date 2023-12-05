package server

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	l "github.com/swmh/gopetbin/internal/logger"
)

type Paste struct {
	BurnAfter int
	Expire    time.Duration
	Content   []byte
}

var errBadValue = errors.New("bad value")

func (s *Server) parseRequest(form *multipart.Form) (Paste, error) {
	var expire time.Duration
	var err error

	v, ok := form.Value["expire"]
	if ok {
		expire, err = time.ParseDuration(v[0])
		if err != nil {
			return Paste{}, errors.Join(err, errBadValue)
		}

		if expire <= 0 {
			return Paste{}, errBadValue
		}
	}

	var burn int

	v, ok = form.Value["burn"]
	if ok {
		burn, err = strconv.Atoi(v[0])
		if err != nil {
			return Paste{}, errors.Join(err, errBadValue)
		}

		if burn < 0 {
			return Paste{}, errBadValue
		}
	}

	var content []byte

	c, ok := form.Value["content"]
	if ok {
		content = []byte(c[0])
	} else {
		files, ok := form.File["content"]
		if !ok {
			return Paste{}, errBadValue
		}

		f, err := files[0].Open()
		if err != nil {
			return Paste{}, err
		}

		content, err = io.ReadAll(f)
		if err != nil {
			return Paste{}, err
		}
	}

	return Paste{
		Expire:    expire,
		BurnAfter: burn,
		Content:   content,
	}, nil
}

func (s *Server) NewUploadForm(logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, s.maxSize)

		err := r.ParseMultipartForm(s.maxFileMemory)
		if err != nil {
			if errors.Is(err, multipart.ErrMessageTooLarge) {
				http.Error(w, "Message Too Large", http.StatusRequestEntityTooLarge)
				return
			}

			var largeErr *http.MaxBytesError
			if errors.As(err, &largeErr) {
				http.Error(w, "Message Too Large", http.StatusRequestEntityTooLarge)
				return
			}

			internalError(w)
			s.logger.Error("Cannot parse multipart form", l.ErrorAttr(err))

			return
		}

		defer r.MultipartForm.RemoveAll()

		paste, err := s.parseRequest(r.MultipartForm)
		if err != nil {
			if errors.Is(err, errBadValue) {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			internalError(w)
			s.logger.Error("Cannot parse request", l.ErrorAttr(err))

			return
		}

		id, err := s.service.CreatePaste(r.Context(), paste)
		if err != nil {
			internalError(w)
			s.logger.Error("Cannot create paste", l.ErrorAttr(err))

			return
		}

		result := fmt.Sprintf("%s/%s", s.publicPath, id)
		w.Write([]byte(result))
	}
}
