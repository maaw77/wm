package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"wm/internal/dto"
	"wm/internal/storage"
)

func TestCheckLinks_OK(t *testing.T) {
	// available: локальный сервер всегда отвечает 200
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(okSrv.Close)

	u, err := url.Parse(okSrv.URL)
	if err != nil {
		t.Fatalf("parse ok server url: %v", err)
	}
	okHostPort := u.Host // "127.0.0.1:12345" (без схемы)

	// not available: домен в зоне .invalid (зарезервирована, не должна резолвиться)
	bad := "nonexistent.invalid"

	st := storage.NewStorage()
	s := NewLinksServer(st)

	body, _ := json.Marshal(dto.CheckLinksRequest{
		Links: []string{okHostPort, bad},
	})

	req := httptest.NewRequest(http.MethodPost, "/links", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	s.CheckLinksHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	var resp dto.CheckLinksResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}

	if resp.LinksNum != 1 {
		t.Fatalf("expected links_num=1, got %d", resp.LinksNum)
	}
	if resp.Links[okHostPort] != "available" {
		t.Fatalf("expected %q=available, got %q", okHostPort, resp.Links[okHostPort])
	}
	if resp.Links[bad] != "not available" {
		t.Fatalf("expected %q=not available, got %q", bad, resp.Links[bad])
	}

	// проверяем, что сохранилось в storage
	task, err := st.Get(resp.LinksNum)
	if err != nil {
		t.Fatalf("storage.Get: %v", err)
	}
	if task.Results[okHostPort] != "available" {
		t.Fatalf("storage expected %q=available, got %q", okHostPort, task.Results[okHostPort])
	}
	if task.Results[bad] != "not available" {
		t.Fatalf("storage expected %q=not available, got %q", bad, task.Results[bad])
	}
}
