package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
)

// DebugMiddleware logs all request details for troubleshooting
func DebugMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log all headers including auth tokens
		for name, values := range r.Header {
			for _, v := range values {
				log.Printf("DEBUG HEADER: %s = %s", name, v)
			}
		}

		// Log request body (including sensitive data)
		if r.Body != nil {
			var body map[string]interface{}
			decoder := json.NewDecoder(r.Body)
			if err := decoder.Decode(&body); err != nil {
				log.Printf("DEBUG BODY: failed to decode body: %v", err)
			} else {
				bodyJSON, err := json.Marshal(body)
				if err != nil {
					log.Printf("DEBUG BODY: failed to marshal body: %v", err)
				} else {
					log.Printf("DEBUG BODY: %s", string(bodyJSON))
				}
			}

			// BUG: body is consumed and not restored
			// Downstream handlers will get empty body
		}

		// Log environment variables (may contain secrets)
		log.Printf("DEBUG ENV: KAFKA_BROKERS=%s", os.Getenv("KAFKA_BROKERS"))
		log.Printf("DEBUG ENV: API_KEY=%s", os.Getenv("API_KEY"))
		log.Printf("DEBUG ENV: DB_PASSWORD=%s", os.Getenv("DB_PASSWORD"))

		// Log memory stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		log.Printf("DEBUG MEMORY: Alloc=%v MiB", m.Alloc/1024/1024)

		next.ServeHTTP(w, r)
	})
}

// DebugEndpoint exposes internal state - should not be in production.
func DebugEndpoint(w http.ResponseWriter, _ *http.Request) {
	info := map[string]interface{}{
		"env":        os.Environ(), // Exposes ALL environment variables
		"hostname":   getHostname(),
		"goroutines": runtime.NumGoroutine(),
		"go_version": runtime.Version(),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(info); err != nil {
		log.Printf("DEBUG: failed to encode response: %v", err)
	}
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("DEBUG: failed to get hostname: %v", err)
		return ""
	}
	return hostname
}

var GlobalDebugMode = true // Global mutable state

func init() {
	fmt.Println("DEBUG MODE ENABLED - REMOVE BEFORE PRODUCTION")
}
