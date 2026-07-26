package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/yuchanns/dnsmasq-dashboard/assets"
	"github.com/yuchanns/dnsmasq-dashboard/internal/dashboard"
)

type Server struct {
	service *dashboard.Service
	logger  *slog.Logger
	static  fs.FS
}

func New(service *dashboard.Service, logger *slog.Logger) (*Server, error) {
	staticFS, err := fs.Sub(assets.FS, "dist")
	if err != nil {
		return nil, fmt.Errorf("open embedded assets: %w", err)
	}
	return &Server{
		service: service,
		logger:  logger,
		static:  staticFS,
	}, nil
}

func (server *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/snapshot", server.snapshot)
	mux.HandleFunc("GET /api/v1/events", server.events)
	mux.HandleFunc("GET /api/v1/healthz", server.health)
	mux.HandleFunc("/", server.frontend)

	return server.securityHeaders(server.requestLog(mux))
}

func (server *Server) snapshot(writer http.ResponseWriter, _ *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	writeJSON(writer, http.StatusOK, server.service.Current())
}

func (server *Server) events(writer http.ResponseWriter, request *http.Request) {
	flusher, ok := writer.(http.Flusher)
	if !ok {
		http.Error(writer, "streaming is not supported", http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache, no-transform")
	writer.Header().Set("Connection", "keep-alive")
	writer.Header().Set("X-Accel-Buffering", "no")

	ticker := time.NewTicker(time.Second)
	heartbeat := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()

	revision := ""
	send := func() bool {
		snapshot := server.service.Current()
		if snapshot.Revision == revision {
			return true
		}
		data, err := json.Marshal(snapshot)
		if err != nil {
			return false
		}
		if _, err := fmt.Fprintf(writer, "event: snapshot\ndata: %s\n\n", data); err != nil {
			return false
		}
		revision = snapshot.Revision
		flusher.Flush()
		return true
	}

	if !send() {
		return
	}
	for {
		select {
		case <-request.Context().Done():
			return
		case <-ticker.C:
			if !send() {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(writer, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (server *Server) health(writer http.ResponseWriter, _ *http.Request) {
	snapshot := server.service.Current()
	status := http.StatusOK
	if !snapshot.Healthy {
		status = http.StatusServiceUnavailable
	}
	writeJSON(writer, status, map[string]any{
		"status":      map[bool]string{true: "ok", false: "degraded"}[snapshot.Healthy],
		"revision":    snapshot.Revision,
		"generatedAt": snapshot.GeneratedAt,
		"warnings":    snapshot.Warnings,
	})
}

func (server *Server) frontend(writer http.ResponseWriter, request *http.Request) {
	requestPath := path.Clean(strings.TrimPrefix(request.URL.Path, "/"))
	if requestPath == "." {
		requestPath = "index.html"
	}

	data, err := fs.ReadFile(server.static, requestPath)
	if err != nil {
		requestPath = "index.html"
		data, err = fs.ReadFile(server.static, requestPath)
		if err != nil {
			http.Error(writer, "frontend unavailable", http.StatusInternalServerError)
			return
		}
	}

	contentType := mime.TypeByExtension(path.Ext(requestPath))
	if contentType != "" {
		writer.Header().Set("Content-Type", contentType)
	}
	if requestPath == "index.html" {
		writer.Header().Set("Cache-Control", "no-cache")
	} else {
		writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	}
	_, _ = writer.Write(data)
}

func (server *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Security-Policy", "default-src 'self'; connect-src 'self'; img-src 'self' data:; style-src 'self'; script-src 'self'; font-src 'self'; base-uri 'none'; frame-ancestors 'none'")
		writer.Header().Set("Referrer-Policy", "no-referrer")
		writer.Header().Set("X-Content-Type-Options", "nosniff")
		writer.Header().Set("X-Frame-Options", "DENY")
		next.ServeHTTP(writer, request)
	})
}

func (server *Server) requestLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		started := time.Now()
		next.ServeHTTP(writer, request)
		if !strings.HasSuffix(request.URL.Path, "healthz") {
			server.logger.Debug("http request", "method", request.Method, "path", request.URL.Path, "duration", time.Since(started))
		}
	})
}

func writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}
