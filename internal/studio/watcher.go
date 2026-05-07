package studio

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"theartifact-cli/internal/api"
	"theartifact-cli/internal/ui"
)

const watchDebounce = 500 * time.Millisecond

// InputWatcher watches .artifact/input/ and auto-ingests new image files.
type InputWatcher struct {
	client       *api.Client
	getWorkspace func() string // dynamic: workspace may change mid-session
	tracker      *JobTracker
	printSystem  func(string)
	printPrompt  func()

	ingested sync.Map // path → struct{}: files sent this session
	stopCh   chan struct{}
}

func newInputWatcher(
	client *api.Client,
	getWorkspace func() string,
	tracker *JobTracker,
	printSystem func(string),
	printPrompt func(),
) *InputWatcher {
	return &InputWatcher{
		client:       client,
		getWorkspace: getWorkspace,
		tracker:      tracker,
		printSystem:  printSystem,
		printPrompt:  printPrompt,
		stopCh:       make(chan struct{}),
	}
}

// Start begins watching dir in a background goroutine.
func (w *InputWatcher) Start(dir string) error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watcher init: %w", err)
	}
	if err := fsw.Add(dir); err != nil {
		fsw.Close()
		return fmt.Errorf("cannot watch %s: %w", dir, err)
	}
	go w.loop(fsw, dir)
	return nil
}

// Stop shuts down the background goroutine.
func (w *InputWatcher) Stop() {
	close(w.stopCh)
}

// ForceIngest ingests all image files in dir, ignoring the already-ingested set.
func (w *InputWatcher) ForceIngest(dir string) {
	wsID := w.getWorkspace()
	if wsID == "" {
		w.printSystem(ui.WarnStyle.Render("No workspace set — use /workspace <id> first."))
		return
	}
	files := imageFiles(dir)
	if len(files) == 0 {
		w.printSystem(ui.MutedStyle.Render("No image files found in .artifact/input/"))
		return
	}
	w.runIngest(wsID, files)
}

func (w *InputWatcher) loop(fsw *fsnotify.Watcher, dir string) {
	defer fsw.Close()

	timer := time.NewTimer(0)
	<-timer.C // drain initial tick so it doesn't fire immediately

	for {
		select {
		case <-w.stopCh:
			return

		case event, ok := <-fsw.Events:
			if !ok {
				return
			}
			if (event.Has(fsnotify.Create) || event.Has(fsnotify.Write)) && isImageFile(event.Name) {
				timer.Reset(watchDebounce)
			}

		case err, ok := <-fsw.Errors:
			if !ok {
				return
			}
			w.printSystem(ui.WarnStyle.Render("Watcher error: " + err.Error()))

		case <-timer.C:
			w.triggerIngest(dir)
		}
	}
}

func (w *InputWatcher) triggerIngest(dir string) {
	wsID := w.getWorkspace()
	if wsID == "" {
		return
	}

	files := w.newFiles(dir)
	if len(files) == 0 {
		return
	}

	w.printSystem(fmt.Sprintf(
		"%s  Detected %d new file(s) in input/ — ingesting…",
		ui.AccentStyle.Render("↑"),
		len(files),
	))
	w.runIngest(wsID, files)
}

// newFiles returns image files in dir not yet ingested this session.
func (w *InputWatcher) newFiles(dir string) []string {
	var out []string
	for _, path := range imageFiles(dir) {
		if _, seen := w.ingested.Load(path); !seen {
			out = append(out, path)
		}
	}
	return out
}

func (w *InputWatcher) runIngest(workspaceID string, files []string) {
	httpClient := &http.Client{}
	var sourceTokens []string

	// Group by mime type so each upload batch is homogeneous
	byMime := groupByMime(files)
	for mime, group := range byMime {
		tokens, err := w.client.RequestUploadURLs(len(group), mime)
		if err != nil {
			w.printSystem(ui.ErrorStyle.Render("✗ Ingest failed (upload URLs): " + err.Error()))
			w.printPrompt()
			return
		}

		for i, path := range group {
			data, err := os.ReadFile(path)
			if err != nil {
				w.printSystem(ui.ErrorStyle.Render("✗ Cannot read " + filepath.Base(path) + ": " + err.Error()))
				w.printPrompt()
				return
			}

			req, _ := http.NewRequest("PUT", tokens[i].URL, bytes.NewReader(data))
			req.Header.Set("Content-Type", mime)

			resp, err := httpClient.Do(req)
			if err != nil {
				w.printSystem(ui.ErrorStyle.Render("✗ Upload failed: " + filepath.Base(path)))
				w.printPrompt()
				return
			}
			resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				w.printSystem(ui.ErrorStyle.Render(fmt.Sprintf("✗ Upload rejected (HTTP %d): %s", resp.StatusCode, filepath.Base(path))))
				w.printPrompt()
				return
			}
			sourceTokens = append(sourceTokens, tokens[i].UploadToken)
		}
	}

	jobID, err := w.client.TriggerIngest(api.IngestRequest{
		WorkspaceID: workspaceID,
		Sources:     sourceTokens,
	})
	if err != nil {
		w.printSystem(ui.ErrorStyle.Render("✗ Ingest trigger failed: " + err.Error()))
		w.printPrompt()
		return
	}

	for _, f := range files {
		w.ingested.Store(f, struct{}{})
	}

	w.tracker.Enqueue(jobID, fmt.Sprintf("ingest %d file(s)", len(files)))
	w.printSystem(fmt.Sprintf(
		"%s  Ingest queued  %s",
		ui.AccentStyle.Render("⦿"),
		ui.MutedStyle.Render(shortID(jobID)+" — processing in background…"),
	))
	w.printPrompt()
}

// ── helpers ───────────────────────────────────────────────────────────────────

func imageFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			path := filepath.Join(dir, e.Name())
			if isImageFile(path) {
				out = append(out, path)
			}
		}
	}
	return out
}

func isImageFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp":
		return true
	}
	return false
}

func groupByMime(files []string) map[string][]string {
	out := make(map[string][]string)
	for _, f := range files {
		m := mimeFromExt(f)
		out[m] = append(out[m], f)
	}
	return out
}

func mimeFromExt(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	default:
		return "image/png"
	}
}
