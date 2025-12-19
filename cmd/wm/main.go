package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync/atomic"
	"syscall"
	"time"

	"wm/config"
	"wm/internal/service"
	"wm/internal/storage"
)

const (
	dataDir         = "data"
	storageFileName = "links.json"
)

var shuttingDown atomic.Bool

func rejectWhenShuttingDown(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if shuttingDown.Load() {
			http.Error(w, "HTTP server is shutting down", http.StatusServiceUnavailable)
			return
		}
		next(w, r)
	}
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	config.InitConfig("./config/config.yaml")

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

	httpClient := config.NewConfiguredHTTPClient()

	srv := service.NewLinksServer(store, httpClient)

	mux := http.NewServeMux()
	mux.HandleFunc("/links", rejectWhenShuttingDown(srv.CheckLinksHandler))
	mux.HandleFunc("/report", rejectWhenShuttingDown(srv.ReportLinksHandler))

	httpServer := config.NewConfiguredHTTPServer(mux)
	errCh := make(chan error, 1)
	go func() {
		slog.Info("HTTP server started", slog.String("addr", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	var sig os.Signal
	select {
	case sig = <-sigCh:
	case err := <-errCh:
		if err != nil {
			slog.Error("HTTP server error", slog.Any("err", err))
		}
	}

	// 1) Переходим в режим “остановки”: новые запросы сразу отклоняем с 503.
	shuttingDown.Store(true)

	var pendingTasks uint64
	store.GetAll(func(id uint64, _ storage.LinkStatus) bool {
		pendingTasks++
		return true
	})

	signalStr := ""
	if sig != nil {
		signalStr = sig.String()
	}

	slog.Info("Graceful shutdown initiated",
		slog.Uint64("pendingtasks", pendingTasks),
		slog.String("signal", signalStr),
	)

	// 2) Перестаём принимать новые соединения и ждём завершения активных handler-ов.
	shutdownTimeout := config.GetConfiguredShutdownTimeout()
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	shutdownStart := time.Now()
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("HTTP server shutdown failed", slog.Any("err", err))
	} else {
		slog.Info("HTTP server stopped")
	}

	// 3) Сохраняем состояние на диск.
	if err := store.Dump(storagePath); err != nil {
		slog.Error("Failed to dump storage state", slog.String("path", storagePath), slog.Any("err", err))
	} else {
		slog.Info("Storage state dumped", slog.String("path", storagePath))
	}

	slog.Info("Shutdown complete",
		slog.Int64("durationms", time.Since(shutdownStart).Milliseconds()),
	)
}
