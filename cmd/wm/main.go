package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"wm/internal/service"
	"wm/internal/storage"
)

const (
	addr            = ":8080"
	dataDir         = "data"
	storageFileName = "links.json"
	shutdownTimeout = 15 * time.Second
)

// main запускает HTTP-сервер и корректно завершает работу с сохранением состояния.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	store := storage.NewStorage()

	storagePath := filepath.Join(dataDir, storageFileName)
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		slog.Error("Failed to create data directory", slog.String("dir", dataDir), slog.Any("err", err))
		os.Exit(1)
	}

	if err := store.Upload(storagePath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("Storage file not found, starting with empty state", slog.String("path", storagePath))
		} else {
			slog.Error("Failed to upload storage state", slog.String("path", storagePath), slog.Any("err", err))
		}
	} else {
		slog.Info("Storage state uploaded", slog.String("path", storagePath))
	}

	srv := service.NewLinksServer(store, &http.Client{Timeout: 5 * time.Second})

	mux := http.NewServeMux()
	mux.HandleFunc("/links", srv.CheckLinksHandler)
	mux.HandleFunc("/report", srv.ReportLinksHandler)

	httpServer := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server started", slog.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		slog.Info("Graceful shutdown initiated", slog.String("signal", sig.String()))
	case err := <-errCh:
		if err != nil {
			slog.Error("HTTP server error", slog.Any("err", err))
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	shutdownStart := time.Now()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown failed", slog.Any("err", err))
	} else {
		slog.Info("HTTP server stopped")
	}

	if err := store.Dump(storagePath); err != nil {
		slog.Error("Failed to dump storage state", slog.String("path", storagePath), slog.Any("err", err))
	} else {
		slog.Info("Storage state dumped", slog.String("path", storagePath))
	}

	slog.Info("Shutdown complete", slog.Int64("duration_ms", time.Since(shutdownStart).Milliseconds()))
}
