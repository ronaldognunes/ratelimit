package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-chi/chi"
	"github.com/joho/godotenv"
	"github.com/ronaldognunes/ratelimiter/internal/database"
	"github.com/ronaldognunes/ratelimiter/internal/limiter"
	"github.com/ronaldognunes/ratelimiter/middleware"
)

func load() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file", err)
	}
}

func main() {
	load()
	limitPerSecToken, _ := strconv.Atoi(os.Getenv("QUANTIDADE_REQUESTS_POR_TOKEN"))
	limitPerSecIp, _ := strconv.Atoi(os.Getenv("QUANTIDADE_REQUESTS_POR_IP"))
	blockDuration, _ := strconv.Atoi(os.Getenv("TEMPO_BLOQUEIO"))
	redisAddr := os.Getenv("REDIS_ADDR")

	fmt.Println(limitPerSecToken)
	fmt.Println(limitPerSecIp)
	fmt.Println(blockDuration)

	bd := database.NewRedisStore(redisAddr)

	limiter := limiter.NewLimiter(bd, limitPerSecIp, limitPerSecToken, blockDuration)

	router := chi.NewRouter()
	router.Use(middleware.RateLimitMiddleware(limiter))
	router.HandleFunc("/teste", func(w http.ResponseWriter, r *http.Request) {
		ip := getClientIP(r)
		token := r.Header.Get("API_KEY")
		w.Write([]byte(fmt.Sprintf(" IP: %s, Token: %s", ip, token)))
	})

	http.ListenAndServe(":8080", router)

}

func getClientIP(r *http.Request) string {
	// Primeiro, verificamos se há proxies ou balanceadores de carga
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0]) // Pega o primeiro IP da lista
	}

	// Em alguns casos, o header X-Real-IP também pode conter o IP correto
	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return ip
	}

	// Se não houver proxy, pegamos o IP do RemoteAddr
	ip = r.RemoteAddr
	if colon := strings.LastIndex(ip, ":"); colon != -1 {
		return ip[:colon] // Remove a porta
	}

	return ip

}
