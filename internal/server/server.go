package server

import (
	"context"
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/bcc-media/clickdown/internal/clickup"
	"github.com/bcc-media/clickdown/internal/db"
	syncworker "github.com/bcc-media/clickdown/internal/sync"
)

type Server struct {
	Store  *db.Store
	Client *clickup.Client
	Sync   *syncworker.Worker
	Web    fs.FS
}

func New(s *db.Store, c *clickup.Client, sw *syncworker.Worker, web embed.FS) *Server {
	sub, err := fs.Sub(web, "web/dist")
	if err != nil {
		sub = web
	}
	return &Server{Store: s, Client: c, Sync: sw, Web: sub}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// API
	mux.HandleFunc("GET /api/tasks", s.listTasks)
	mux.HandleFunc("PATCH /api/tasks/{id}", s.patchTask)
	mux.HandleFunc("PUT /api/tasks/{id}/tags", s.putTaskTags)
	mux.HandleFunc("GET /api/tasks/{id}/comments", s.listTaskComments)
	mux.HandleFunc("POST /api/tasks/{id}/comments", s.postTaskComment)
	mux.HandleFunc("GET /api/statuses", s.listStatuses)
	mux.HandleFunc("GET /api/tags", s.listTags)
	mux.HandleFunc("POST /api/sync", s.triggerSync)
	mux.HandleFunc("GET /api/sync/status", s.syncStatus)
	mux.HandleFunc("GET /api/settings", s.getSettings)
	mux.HandleFunc("PATCH /api/settings", s.patchSettings)

	// SPA: serve embedded assets; everything else falls through to index.html.
	mux.Handle("/", spaHandler(s.Web))

	return logRequests(mux)
}

func (s *Server) Start(ctx context.Context, addr string) error {
	srv := &http.Server{Addr: addr, Handler: s.Handler()}
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeoutCause(context.Background(), 0, nil)
		_ = cancel
		_ = shutdownCtx
		return srv.Close()
	case err := <-errCh:
		return err
	}
}

func spaHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(root, path); err != nil {
			// Fallback to index.html for SPA routing.
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func logRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
}
