package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"watch-gateway/internal/bus"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config.yaml")
	flag.Parse()

	cfg, err := bus.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config failed: %v", err)
	}

	client := bus.NewClient()
	service := bus.NewService(client)
	barkClient := bus.NewBarkClientFromEnv()
	limiter := bus.NewRateLimiter(4, time.Minute)

	mux := http.NewServeMux()
	bus.RegisterRoutes(mux, cfg, service, barkClient, limiter)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"code":0,"message":"ok"}`))
	})

	server := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("watch-gateway started on %s (config=%s)", cfg.Server.Addr, *configPath)

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server stopped: %v", err)
	}
}
