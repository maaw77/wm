// Package main запускает HTTP-сервис проверки доступности ссылок.
//
// Эндпоинты:
// - POST /links: проверка ссылок и выдача статусов + номер набора.
// - POST /report: генерация PDF-отчета по ранее созданным наборам.
//
// Поддерживается graceful shutdown: при остановке новые запросы получают 503,
// затем сервер корректно завершается и состояние in-memory storage сохраняется на диск.
package main

import (
	"context"
	"errors"
	"flag"
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

// shuttingDown включает режим остановки: новые запросы отклоняются с 503.
var shuttingDown atomic.Bool

// rejectWhenShuttingDown отклоняет запросы с 503, если начат graceful shutdown.
func rejectWhenShuttingDown(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if shuttingDown.Load() {
			http.Error(w, "HTTP server is shutting down", http.StatusServiceUnavailable)
			return
		}
		next(w, r)
	}
}

// main запускает сервер, ожидает SIGINT/SIGTERM и при остановке сохраняет состояние storage.
//
// Конфигурация задается флагом -config (по умолчанию "./config/config.yaml") и
// используется для настройки HTTP-сервера/клиента, таймаута graceful shutdown,
// а также пути к файлу состояния storage.
func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	var pathConfig string

	flag.StringVar(&pathConfig, "config", "./config/config.yaml", "the path to the configuration file")
	flag.Parse()
	config.InitConfig(pathConfig)

	store := storage.NewStorage()
	storagePath := filepath.Join(config.GetConfiguredStorageDataDir(), config.GetConfiguredStorageFileName())

	if err := os.MkdirAll(config.GetConfiguredStorageDataDir(), 0o755); err != nil {
		slog.Error("Failed to create data directory", slog.String("dir", config.GetConfiguredStorageDataDir()), slog.Any("err", err))
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
