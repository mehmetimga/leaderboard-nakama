package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// NewRouter creates a new HTTP router with all routes configured
func NewRouter(handler *Handler) *chi.Mux {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(corsMiddleware)

	// Health check
	r.Get("/health", handler.HealthCheck)

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		// Score submission
		r.Post("/scores", handler.SubmitScore)

		// Leaderboard operations
		r.Route("/leaderboards/{id}", func(r chi.Router) {
			r.Get("/", handler.GetLeaderboard)
			r.Get("/around/{userId}", handler.GetAroundUser)
			r.Get("/users/{userId}", handler.GetUserRank)
			r.Delete("/users/{userId}", handler.DeleteScore)
		})

		// Snapshot operations (admin)
		r.Post("/snapshots", handler.CreateSnapshot)
	})

	return r
}

// corsMiddleware adds CORS headers for development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, X-User-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

