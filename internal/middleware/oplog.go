package middleware

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OplogUser struct {
	UserID   int64
	Username string
}

type UserExtractor func(ctx context.Context) *OplogUser

type responseRecorder struct {
	http.ResponseWriter
	statusCode int
	body       bytes.Buffer
}

func (r *responseRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseRecorder) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func OperationLog(pool *pgxpool.Pool, extractor UserExtractor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if pool == nil {
				next.ServeHTTP(w, r)
				return
			}

			if r.Method == http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}

			start := time.Now()

			var reqBody string
			if r.Body != nil {
				bodyBytes, _ := io.ReadAll(r.Body)
				reqBody = truncate(string(bodyBytes), 4096)
				r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			}

			rec := &responseRecorder{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(rec, r)

			duration := time.Since(start).Milliseconds()

			var userID *int64
			var username string
			if extractor != nil {
				if u := extractor(r.Context()); u != nil {
					userID = &u.UserID
					username = u.Username
				}
			}
			if username == "" {
				username = "-"
			}

			respBody := truncate(rec.body.String(), 4096)

			go func() {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()

				_, err := pool.Exec(ctx, `
					INSERT INTO operation_logs (user_id, username, method, path, status_code, duration_ms, client_ip, request_body, response_body)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
					userID, username, r.Method, r.URL.Path, rec.statusCode, duration, r.RemoteAddr, reqBody, respBody)
				if err != nil {
					slog.Error("failed to write operation log", "error", err)
				}
			}()
		})
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
