package httpapi

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"convertthis/internal/convert"
	"convertthis/internal/job"
	"convertthis/internal/storage"
)

type Server struct {
	queue     job.Queue
	store     storage.Storage
	static    http.FileSystem
	index     []byte
	maxUpload int64
}

func New(q job.Queue, store storage.Storage, static http.FileSystem, index []byte, maxUpload int64) *Server {
	return &Server{queue: q, store: store, static: static, index: index, maxUpload: maxUpload}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/convert", s.handleConvert)
	mux.HandleFunc("GET /api/jobs/{id}", s.handleStatus)
	mux.HandleFunc("GET /api/download/{id}", s.handleDownload)
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(s.static)))
	mux.HandleFunc("GET /{$}", s.handleIndex) // {$} matches the root path exactly, not as a catch-all
	return withLogging(mux)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(s.index)
}

func (s *Server) handleConvert(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.maxUpload)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "file too large or malformed upload")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "missing file")
		return
	}
	defer file.Close()

	to, err := convert.ParseFormat(r.FormValue("target"))
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(header.Filename)), ".")
	from, err := convert.ParseFormat(ext)
	if err != nil {
		writeError(w, http.StatusBadRequest, "unsupported input type: ."+ext)
		return
	}
	if from == to {
		writeError(w, http.StatusBadRequest, "source and target formats are the same")
		return
	}

	base := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))

	j, err := s.queue.Enqueue(r.Context(), job.Request{
		Input: file, From: from, To: to, BaseName: base,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "conversion failed: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"id":       j.ID,
		"status":   j.Status,
		"filename": j.OutputName,
		"download": "/api/download/" + j.ID,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	j, ok := s.queue.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	writeJSON(w, http.StatusOK, j)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	j, ok := s.queue.Get(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "job not found")
		return
	}
	if j.Status != job.StatusDone {
		writeError(w, http.StatusConflict, "job not ready")
		return
	}

	rc, err := s.store.Open(r.Context(), j.OutputKey)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "output unavailable")
		return
	}
	defer rc.Close()

	w.Header().Set("Content-Type", j.To.MIME())
	w.Header().Set("Content-Disposition", "attachment; filename=\""+j.OutputName+"\"")
	_, _ = io.Copy(w, rc)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
