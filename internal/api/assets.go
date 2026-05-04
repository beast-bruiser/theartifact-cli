package api

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// DownloadJobAssets downloads all assets from a succeeded job into destDir.
// Returns the list of file paths written and a slice of per-asset errors (non-fatal).
func DownloadJobAssets(result *JobResult, destDir string) (paths []string, errs []error) {
	for i, a := range result.Assets {
		prefix := "asset"
		if a.ID != "" {
			prefix = a.ID
		}
		name := AssetFilename(prefix, i, a.URL)
		dest := filepath.Join(destDir, name)
		if err := DownloadAsset(a.URL, dest); err != nil {
			errs = append(errs, fmt.Errorf("asset %d: %w", i+1, err))
			continue
		}
		paths = append(paths, dest)
	}
	return paths, errs
}

// DownloadAsset fetches a presigned asset URL and writes it to destPath atomically.
// The URL is presigned — no auth header is sent.
func DownloadAsset(assetURL, destPath string) error {
	resp, err := http.Get(assetURL) //nolint:gosec // presigned URL from trusted API response
	if err != nil {
		return fmt.Errorf("download request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return fmt.Errorf("cannot create output directory: %w", err)
	}

	// Write to a temp file, then rename for atomicity
	tmp := destPath + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("cannot create temp file: %w", err)
	}

	if _, err := io.Copy(f, resp.Body); err != nil {
		f.Close()
		os.Remove(tmp)
		return fmt.Errorf("write failed: %w", err)
	}
	f.Close()

	if err := os.Rename(tmp, destPath); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("cannot move file to destination: %w", err)
	}

	return nil
}

// AssetFilename derives a safe filename for an asset given a job ID, index,
// and the asset's URL. Falls back to .bin when the extension cannot be inferred.
func AssetFilename(jobID string, index int, assetURL string) string {
	ext := extensionFromURL(assetURL)
	short := jobID
	if len(jobID) > 12 {
		short = jobID[len(jobID)-8:]
	}
	return fmt.Sprintf("%s_%d%s", short, index+1, ext)
}

func extensionFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ".bin"
	}
	// Strip query string and get extension from path
	path := u.Path
	ext := filepath.Ext(path)
	if ext != "" {
		return ext
	}

	// Try Content-Type from a HEAD request (best-effort, non-fatal)
	resp, err := http.Head(rawURL) //nolint:gosec
	if err != nil {
		return ".bin"
	}
	defer resp.Body.Close()

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		return ".bin"
	}
	mt, _, _ := mime.ParseMediaType(ct)
	switch mt {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	default:
		// e.g. "image/png" → "png"
		parts := strings.SplitN(mt, "/", 2)
		if len(parts) == 2 && parts[1] != "" {
			return "." + parts[1]
		}
		return ".bin"
	}
}
