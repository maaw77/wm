package service

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"wm/internal/dto"
	"wm/internal/storage"
)

const maxRequestBodyBytes = 1 << 20 // 1MB

// linksServer инкапсулирует зависимости HTTP-обработчиков.
type linksServer struct {
	store  storage.Storage
	client *http.Client
}

// NewLinksServer создает сервер с обработчиками для работы со ссылками.
func NewLinksServer(store storage.Storage, clnt *http.Client) *linksServer {
	if clnt == nil {
		clnt = &http.Client{Timeout: 5 * time.Second} // или создаем дефолтный
	}
	return &linksServer{store: store, client: clnt}
}

// CheckLinksHandler принимает список ссылок, проверяет их и возвращает статусы + номер набора.
func (s *linksServer) CheckLinksHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Защита от слишком больших JSON.
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req dto.CheckLinksRequest
	if err := dec.Decode(&req); err != nil {
		slog.Warn("Invalid request JSON", slog.Any("err", err))
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(req.Links) == 0 {
		http.Error(w, "links must not be empty", http.StatusBadRequest)
		return
	}

	taskID, err := s.store.Add(req.Links)
	if err != nil {
		if errors.Is(err, storage.ErrEmptyInpData) {
			http.Error(w, "links must not be empty", http.StatusBadRequest)
			return
		}
		slog.Error("Failed to add batch to storage", slog.Any("err", err))
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	slog.Info(
		"Batch processing started",
		slog.Uint64("task_id", taskID),
		slog.Int("links_count", len(req.Links)),
	)

	// Реальная проверка ссылок.
	// client := &http.Client{Timeout: 5 * time.Second}

	statuses := make(map[string]string, len(req.Links))
	for _, raw := range req.Links {
		linkStart := time.Now()

		u := raw
		// Если схема не указана, добавляем http:// чтобы net/http понял URL.
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			u = "http://" + u
		}

		status := "not available"

		// reqGet, _ := http.NewRequest(http.MethodGet, u, nil)
		resp, err := s.client.Get(u)
		if err != nil {
			slog.Warn(
				"Link check error",
				slog.Uint64("task_id", taskID),
				slog.String("link", raw),
				slog.Any("err", err),
			)
		} else {
			_ = resp.Body.Close()
			if resp.StatusCode < 400 {
				status = "available"
			}
		}

		statuses[raw] = status

		slog.Info(
			"Link check completed",
			slog.Uint64("task_id", taskID),
			slog.String("link", raw),
			slog.String("status", status),
			slog.Int64("duration_ms", time.Since(linkStart).Milliseconds()),
		)
	}

	// Сохраняем результаты в in-memory storage
	if err := s.store.UpdateResults(taskID, statuses); err != nil {
		slog.Error(
			"Failed to update results in storage",
			slog.Uint64("task_id", taskID),
			slog.Any("err", err),
		)
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := dto.CheckLinksResponse{
		Links:    statuses,
		LinksNum: taskID,
	}

	slog.Info(
		"Batch processing completed",
		slog.Uint64("task_id", taskID),
		slog.Int("links_count", len(req.Links)),
		slog.Int64("duration_ms", time.Since(start).Milliseconds()),
	)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// ReportLinksHandler читает список номеров задач и в будущем будет формировать PDF-отчет.
// Сейчас только валидирует запрос и возвращает 501 Not Implemented.
func (s *linksServer) ReportLinksHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	if len(req.LinksList) == 0 {
		http.Error(w, "links_list must not be empty", http.StatusBadRequest)
		return
	}

	http.Error(w, "not implemented", http.StatusNotImplemented)
}
