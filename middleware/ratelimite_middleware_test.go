package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ronaldognunes/ratelimiter/internal/database"
	"github.com/ronaldognunes/ratelimiter/internal/limiter"
	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {

	ctx := context.Background()

	qtdTokens := 10
	qtdIp := 10
	blockDuration := 10
	redisAddr := "localhost:6379"

	bd := database.NewRedisStore(redisAddr)

	li := limiter.NewLimiter(bd, qtdIp, qtdTokens, blockDuration)

	middleware := RateLimitMiddleware(li)
	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	server := httptest.NewServer(handler)
	defer server.Close()
	t.Run("Teste de requisição com token", func(t *testing.T) {
		client := &http.Client{}
		var status []int

		for i := 0; i < 11; i++ {
			req, _ := http.NewRequest("GET", server.URL, nil)
			req.Header.Add("API_KEY", "token")
			req.Header.Set("X-Real-IP", "192.168.1.1")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Erro ao fazer requisição: %v", err)
			}
			defer resp.Body.Close()
			status = append(status, resp.StatusCode)
		}

		assert.Equal(t, http.StatusOK, status[0])
		assert.Equal(t, http.StatusOK, status[4])
		assert.Equal(t, http.StatusTooManyRequests, status[10])

		time.Sleep(time.Duration(blockDuration) * time.Second)
		status = []int{}
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("GET", server.URL, nil)
			req.Header.Add("API_KEY", "token")
			req.Header.Set("X-Real-IP", "192.168.1.1")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Erro ao fazer requisição: %v", err)
			}
			defer resp.Body.Close()
			status = append(status, resp.StatusCode)
		}

		for i := range status {
			assert.Equal(t, http.StatusOK, status[i])
		}

	})

	t.Run("Teste de requisição com ip", func(t *testing.T) {
		client := &http.Client{}
		var status []int

		for i := 0; i < 11; i++ {
			req, _ := http.NewRequest("GET", server.URL, nil)
			req.Header.Set("X-Real-IP", "192.168.1.2")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Erro ao fazer requisição: %v", err)
			}
			defer resp.Body.Close()
			status = append(status, resp.StatusCode)
		}

		assert.Equal(t, http.StatusOK, status[0])
		assert.Equal(t, http.StatusOK, status[4])
		assert.Equal(t, http.StatusTooManyRequests, status[10])

		time.Sleep(time.Duration(blockDuration) * time.Second)
		status = []int{}
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("GET", server.URL, nil)
			req.Header.Set("X-Real-IP", "192.168.1.2")

			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Erro ao fazer requisição: %v", err)
			}
			defer resp.Body.Close()
			status = append(status, resp.StatusCode)
		}

		for i := range status {
			assert.Equal(t, http.StatusOK, status[i])
		}
	})

	t.Run("Teste de requisição concorrente com token", func(t *testing.T) {
		client := &http.Client{}
		wg := sync.WaitGroup{}
		result := make(chan int, 11)

		for i := 0; i < 11; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequest("GET", server.URL, nil)
				req.Header.Add("API_KEY", "123")
				req.Header.Set("X-Real-IP", "192.168.1.3")

				resp, err := client.Do(req)
				if err != nil {
					t.Errorf("Erro na requisição concorrente: %v", err)
					return
				}
				defer resp.Body.Close()
				result <- resp.StatusCode
			}()
		}

		wg.Wait()
		close(result)
		var sucesso, erro int
		for r := range result {
			if r == http.StatusOK {
				sucesso++
			} else {
				erro++
			}
		}
		assert.Equal(t, 10, sucesso)
		assert.Equal(t, 1, erro)

		// aguarda o tempo de bloqueio e efetua mais 10 requisições
		time.Sleep(time.Duration(blockDuration) * time.Second)
		result = make(chan int, 10)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequest("GET", server.URL, nil)
				req.Header.Add("API_KEY", "123")
				req.Header.Set("X-Real-IP", "192.168.1.3")

				resp, err := client.Do(req)
				if err != nil {
					t.Errorf("Erro na requisição concorrente: %v", err)
					return
				}
				defer resp.Body.Close()
				result <- resp.StatusCode
			}()
		}
		wg.Wait()
		close(result)
		sucesso, erro = 0, 0
		for r := range result {
			if r == http.StatusOK {
				sucesso++
			} else {
				erro++
			}
		}
		assert.Equal(t, 10, sucesso)
		assert.Equal(t, 0, erro)
	})

	t.Run("Teste de requisição concorrente com ip", func(t *testing.T) {
		client := &http.Client{}
		wg := sync.WaitGroup{}
		results := make(chan int, 11)

		for i := 0; i < 11; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequest("GET", server.URL, nil)
				req.Header.Set("X-Real-IP", "192.168.1.4")

				resp, err := client.Do(req)
				if err != nil {
					t.Errorf("Erro ao fazer requisição: %v", err)
					return
				}
				defer resp.Body.Close()
				results <- resp.StatusCode
			}()
		}
		wg.Wait()
		close(results)
		var sucesso, erro int
		for r := range results {
			if r == http.StatusOK {
				sucesso++
			} else {
				erro++
			}
		}
		assert.Equal(t, 10, sucesso)
		assert.Equal(t, 1, erro)

		// aguarda o tempo de bloqueio e efetua mais 10 requisições
		time.Sleep(time.Duration(blockDuration) * time.Second)
		results = make(chan int, 10)
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				req, _ := http.NewRequest("GET", server.URL, nil)
				req.Header.Set("X-Real-IP", "192.168.1.4")

				resp, err := client.Do(req)
				if err != nil {
					t.Errorf("Erro na requisição concorrente: %v", err)
					return
				}
				defer resp.Body.Close()
				results <- resp.StatusCode
			}()
		}
		wg.Wait()
		close(results)
		sucesso, erro = 0, 0
		for r := range results {
			if r == http.StatusOK {
				sucesso++
			} else {
				erro++
			}
		}
		assert.Equal(t, 10, sucesso)
		assert.Equal(t, 0, erro)

	})

	bd.Client.FlushAll(ctx)
}
