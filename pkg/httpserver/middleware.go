package httpserver

import (
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/ogen-go/ogen/json"
	"github.com/rs/cors"

	"ndbx/pkg/logger"
)

type Middleware = func(http.Handler) http.Handler

func HeartbeatMiddleware(endpoint string) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if (r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.EqualFold(r.URL.Path, endpoint) {
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				data, _ := json.Marshal(map[string]string{"status": "ok"})
				_, _ = w.Write(data)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}

func CORSMiddleware(l cors.Logger) Middleware {
	return cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Authorization"},
		AllowCredentials: true,
		Logger:           l,
	}).Handler
}

func DocsMiddleware(l logger.Interface) Middleware {
	return func(h http.Handler) http.Handler {
		var (
			redocHTML   []byte
			openapiYAML []byte
			swaggerHTML []byte
		)
		once := &sync.Once{}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !loadStatic(once, l, &redocHTML, &openapiYAML, &swaggerHTML) {
				h.ServeHTTP(w, r)
				return
			}

			path := strings.TrimSuffix(r.URL.Path, "/")
			if path == "/api/docs" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(redocHTML)
				return
			}
			if path == "/api/swagger" && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(swaggerHTML)
				return
			}
			if (path == "/api/docs/openapi.yaml" || path == "/api/swagger/openapi.yaml") && (r.Method == http.MethodGet || r.Method == http.MethodHead) {
				w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(openapiYAML)
				return
			}
			h.ServeHTTP(w, r)
		})
	}
}

func Wrap(h http.Handler, middlewares ...Middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}

	return h
}

func loadStatic(
	once *sync.Once,
	l logger.Interface,
	redocHTML *[]byte,
	openapiYAML *[]byte,
	swaggerHTML *[]byte,
) bool {
	once.Do(func() {
		var err error
		*redocHTML, err = os.ReadFile("docs/redoc.html")
		if err != nil {
			l.Errorf("failed to load redoc html: %s", err.Error())
			return
		}
		*openapiYAML, err = os.ReadFile("docs/openapi.yaml")
		if err != nil {
			l.Errorf("failed to load openapi yaml: %s", err.Error())
			return
		}
		*swaggerHTML, err = os.ReadFile("docs/swagger.html")
		if err != nil {
			l.Errorf("failed to load swagger html: %s", err.Error())
			return
		}
	})

	return len(*redocHTML) > 0 && len(*openapiYAML) > 0 && len(*swaggerHTML) > 0
}
