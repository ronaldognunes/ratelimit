package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/ronaldognunes/ratelimiter/internal/limiter"
)

func RateLimitMiddleware(l *limiter.Limiter) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.Background()
			key := getClientIP(r)
			typeLimit := "IP"

			if token := r.Header.Get("API_KEY"); token != "" {
				key = token
				typeLimit = "TOKEN"
			}

			if err := l.AllowRequest(ctx, typeLimit, key); err != nil {
				http.Error(w, err.Error(), http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	ip = r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		return ip[:colon]
	}

	return ip

}
