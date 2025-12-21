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
	okSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(okSrv.Close)

	u, err := url.Parse(okSrv.URL)
	if err != nil {
		t.Fatalf("parse ok server url: %v", err)
	}
	okHostPort := u.Host // "127.0.0.1:12345"

	// not available
	bad := "nonexistent.invalid"

	st := storage.NewStorage()
	s := NewLinksServer(st, nil)

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

	// проверяка storage
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

func TestReportLinks_OK(t *testing.T) {
	st := storage.NewStorage()
	s := NewLinksServer(st, nil)

	id1, err := st.Add([]string{"google.com"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := st.UpdateResults(id1, map[string]string{"google.com": "available"}); err != nil {
		t.Fatalf("UpdateResults: %v", err)
	}

	id2, err := st.Add([]string{"malformedlink.gg"})
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := st.UpdateResults(id2, map[string]string{"malformedlink.gg": "not available"}); err != nil {
		t.Fatalf("UpdateResults: %v", err)
	}

	body, _ := json.Marshal(dto.ReportRequest{LinksList: []uint64{id1, id2}})
	req := httptest.NewRequest(http.MethodPost, "/report", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	s.ReportLinksHandler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d, body=%s", http.StatusOK, rr.Code, rr.Body.String())
	}

	if ct := rr.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("expected Content-Type application/pdf, got %q", ct)
	}
	if cd := rr.Header().Get("Content-Disposition"); cd == "" {
		t.Fatalf("expected Content-Disposition, got empty")
	}

	b := rr.Body.Bytes()
	if len(b) < 4 {
		t.Fatalf("expected non-empty pdf body")
	}
	if string(b[:4]) != "%PDF" {
		t.Fatalf("expected PDF signature %%PDF, got %q", string(b[:4]))
	}
}
