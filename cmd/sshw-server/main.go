package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yinheli/sshw/configserver"
)

func main() {
	address := envOrDefault("SSHW_SERVER_ADDR", ":8080")
	databasePath := envOrDefault("SSHW_DATABASE_PATH", "/data/sshw.db")
	adminUsername := envOrDefault("SSHW_ADMIN_USERNAME", "admin")
	adminPassword, err := loadSecret("SSHW_ADMIN_PASSWORD", "SSHW_ADMIN_PASSWORD_FILE")
	if err != nil {
		log.Fatal(err)
	}
	masterKey, err := loadSecret("SSHW_MASTER_KEY", "SSHW_MASTER_KEY_FILE")
	if err != nil {
		log.Fatal(err)
	}
	sessionSecret := strings.TrimSpace(os.Getenv("SSHW_SESSION_SECRET"))
	if sessionSecret == "" {
		sessionSecret = masterKey + ":session"
	}

	store, err := configserver.OpenStore(databasePath, masterKey)
	if err != nil {
		log.Fatalf("open config store: %v", err)
	}
	defer store.Close()

	app, err := configserver.NewServer(configserver.ServerConfig{
		Store:          store,
		AdminUsername:  adminUsername,
		AdminPassword:  adminPassword,
		SessionSecret:  sessionSecret,
		DefaultProfile: envOrDefault("SSHW_DEFAULT_PROFILE", "default"),
	})
	if err != nil {
		log.Fatalf("create config server: %v", err)
	}

	server := &http.Server{
		Addr:              address,
		Handler:           app.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       90 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("sshw config server listening on %s", address)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func loadSecret(valueName, fileName string) (string, error) {
	if value := strings.TrimSpace(os.Getenv(valueName)); value != "" {
		return value, nil
	}
	if path := strings.TrimSpace(os.Getenv(fileName)); path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("read %s: %w", fileName, err)
		}
		if value := strings.TrimSpace(string(data)); value != "" {
			return value, nil
		}
	}
	return "", fmt.Errorf("%s or %s is required", valueName, fileName)
}
