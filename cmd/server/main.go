package main

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"convertthis/internal/convert"
	"convertthis/internal/httpapi"
	"convertthis/internal/job"
	"convertthis/internal/storage"
	"convertthis/web"
)

func main() {
	addr := env("ADDR", ":8080")
	workDir := env("WORK_DIR", filepath.Join(os.TempDir(), "convertthis"))
	const maxUpload = int64(100 << 20)

	conv := convert.NewFFmpeg()
	if !conv.Available() {
		log.Println("WARNING: ffmpeg not found on PATH — conversions will fail.")
		log.Println("  Install it →  Windows: winget install Gyan.FFmpeg   macOS: brew install ffmpeg   Linux: apt install ffmpeg")
	}

	store, err := storage.NewLocal(workDir)
	if err != nil {
		log.Fatalf("storage: %v", err)
	}

	queue := job.NewSyncQueue(conv, store)

	staticFS, err := fs.Sub(web.Files, "static")
	if err != nil {
		log.Fatal(err)
	}
	index, err := web.Files.ReadFile("index.html")
	if err != nil {
		log.Fatal(err)
	}

	srv := httpapi.New(queue, store, http.FS(staticFS), index, maxUpload)

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           srv.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("convertthis listening on http://localhost%s  (work dir: %s)", addr, workDir)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
