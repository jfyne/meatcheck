package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalFileHandlerServesFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "a.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	h := localFileHandler(tmp)
	req := httptest.NewRequest(http.MethodGet, "/file?path=a.txt", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if rr.Body.String() != "hello" {
		t.Fatalf("expected body hello, got %q", rr.Body.String())
	}
}

func TestLocalFileHandlerBlocksTraversal(t *testing.T) {
	tmp := t.TempDir()
	h := localFileHandler(tmp)
	req := httptest.NewRequest(http.MethodGet, "/file?path=../etc/passwd", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
