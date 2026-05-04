package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadAsset(t *testing.T) {
	body := []byte("fake-image-data")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}))
	defer srv.Close()

	dir := t.TempDir()
	dest := filepath.Join(dir, "out.png")

	if err := DownloadAsset(srv.URL+"/asset.png", dest); err != nil {
		t.Fatalf("DownloadAsset: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(body) {
		t.Errorf("content mismatch: got %q, want %q", got, body)
	}
}

func TestDownloadAsset_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	dir := t.TempDir()
	err := DownloadAsset(srv.URL+"/asset.png", filepath.Join(dir, "out.png"))
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestAssetFilename(t *testing.T) {
	cases := []struct {
		jobID    string
		index    int
		url      string
		wantSufx string
	}{
		{"job_abc123456789", 0, "https://cdn.example.com/img.png?sig=x", ".png"},
		{"job_abc123456789", 1, "https://cdn.example.com/img.jpg", ".jpg"},
		{"job_abc123456789", 0, "https://cdn.example.com/img", ".bin"},
	}

	for _, tc := range cases {
		name := AssetFilename(tc.jobID, tc.index, tc.url)
		if filepath.Ext(name) != tc.wantSufx {
			t.Errorf("AssetFilename(%q, %d, %q) = %q, want ext %q", tc.jobID, tc.index, tc.url, name, tc.wantSufx)
		}
	}
}
