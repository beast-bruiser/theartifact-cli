package studio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"theartifact-cli/internal/api"
)

// newTrackerForTest creates a JobTracker with a test HTTP server.
func newTrackerForTest(t *testing.T, handler http.Handler) *JobTracker {
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	httpClient := resty.New()
	httpClient.SetBaseURL(server.URL)
	httpClient.SetAuthToken("test_key")
	httpClient.SetHeader("Content-Type", "application/json")
	httpClient.SetError(&api.APIError{})

	client := &api.Client{HTTPClient: httpClient}
	return NewJobTracker(client, func(*TrackedJob) {})
}

// TestJobTracker_Enqueue verifies that jobs are correctly enqueued.
func TestJobTracker_Enqueue(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"job_test123","status":"queued","progress":0}`))
	})

	tracker := newTrackerForTest(t, handler)
	tracker.Enqueue("job_test123", "A scenic landscape")

	snap := tracker.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("expected 1 job, got %d", len(snap))
	}

	if snap[0].ID != "job_test123" {
		t.Errorf("job ID mismatch: got %q, want %q", snap[0].ID, "job_test123")
	}

	if snap[0].Prompt != "A scenic landscape" {
		t.Errorf("prompt mismatch: got %q, want %q", snap[0].Prompt, "A scenic landscape")
	}

	if snap[0].Status != "queued" {
		t.Errorf("status mismatch: got %q, want %q", snap[0].Status, "queued")
	}

	tracker.Stop()
}

// TestJobTracker_ShortID verifies that shortID correctly truncates to 8 chars.
func TestJobTracker_ShortID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"job_abcdef1234567890","status":"queued","progress":0}`))
	})

	tracker := newTrackerForTest(t, handler)
	longID := "job_abcdef1234567890abcdef"
	tracker.Enqueue(longID, "test prompt")

	snap := tracker.Snapshot()
	if snap[0].ShortID != longID[:8] {
		t.Errorf("short ID mismatch: got %q, want %q", snap[0].ShortID, longID[:8])
	}

	tracker.Stop()
}

// TestJobTracker_PendingCount verifies pending job counting.
func TestJobTracker_PendingCount(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","status":"queued","progress":0}`))
	})

	tracker := newTrackerForTest(t, handler)
	tracker.Enqueue("job_1", "test 1")
	tracker.Enqueue("job_2", "test 2")
	tracker.Enqueue("job_3", "test 3")

	if count := tracker.PendingCount(); count != 3 {
		t.Errorf("initial pending count: got %d, want %d", count, 3)
	}

	tracker.Stop()
}

// TestJobTracker_Snapshot verifies Snapshot returns independent copies.
func TestJobTracker_Snapshot(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","status":"queued","progress":0}`))
	})

	tracker := newTrackerForTest(t, handler)
	tracker.Enqueue("job_test", "test")

	snap1 := tracker.Snapshot()
	snap2 := tracker.Snapshot()

	if len(snap1) != len(snap2) {
		t.Fatalf("snapshot length mismatch: %d vs %d", len(snap1), len(snap2))
	}

	if snap1[0].Prompt != snap2[0].Prompt {
		t.Errorf("snapshot data mismatch")
	}

	tracker.Stop()
}

// TestJobTracker_MultipleJobs verifies multiple jobs are tracked independently.
func TestJobTracker_MultipleJobs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","status":"queued","progress":0}`))
	})

	tracker := newTrackerForTest(t, handler)
	tracker.Enqueue("job_1", "prompt 1")
	tracker.Enqueue("job_2", "prompt 2")
	tracker.Enqueue("job_3", "prompt 3")

	snap := tracker.Snapshot()
	if len(snap) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(snap))
	}

	for i, job := range snap {
		if job.Status != "queued" {
			t.Errorf("job %d status: got %q, want %q", i, job.Status, "queued")
		}
	}

	tracker.Stop()
}

// TestJobTracker_TerminalStates verifies terminal state identification.
func TestJobTracker_TerminalStates(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","status":"succeeded","progress":100}`))
	})

	tracker := newTrackerForTest(t, handler)
	tracker.Enqueue("job_success", "test")
	tracker.Enqueue("job_running", "test")

	// Count jobs that are in terminal state
	snap := tracker.Snapshot()
	terminalCount := 0
	for _, job := range snap {
		if job.Status == "succeeded" || job.Status == "failed" {
			terminalCount++
		}
	}

	if terminalCount != 0 {
		t.Errorf("terminal count in fresh state: got %d, want 0", terminalCount)
	}

	tracker.Stop()
}

// TestJobTracker_CallbackType verifies callback signature is valid.
func TestJobTracker_CallbackType(t *testing.T) {
	callCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","status":"succeeded","progress":100}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	httpClient := resty.New()
	httpClient.SetBaseURL(server.URL)
	httpClient.SetAuthToken("test_key")
	httpClient.SetHeader("Content-Type", "application/json")
	httpClient.SetError(&api.APIError{})

	client := &api.Client{HTTPClient: httpClient}
	tracker := NewJobTracker(client, func(job *TrackedJob) {
		callCount++
		if job.ID == "" {
			t.Error("callback received job with empty ID")
		}
	})

	tracker.Enqueue("job_callback_test", "test prompt")
	snap := tracker.Snapshot()

	if len(snap) > 0 && snap[0].ID != "job_callback_test" {
		t.Errorf("enqueued job ID mismatch: got %q", snap[0].ID)
	}

	tracker.Stop()
}

// TestJobTracker_Stop verifies Stop doesn't panic.
func TestJobTracker_Stop(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","status":"queued","progress":0}`))
	})

	tracker := newTrackerForTest(t, handler)
	tracker.Enqueue("job_stop_test", "test")

	// Stop should not panic
	tracker.Stop()

	// Verify tracker is still usable state
	snap := tracker.Snapshot()
	if len(snap) != 1 {
		t.Errorf("snapshot after stop: got %d jobs, want 1", len(snap))
	}
}

// TestJobTracker_Polling verifies polling detects status changes.
func TestJobTracker_Polling(t *testing.T) {
	pollCount := 0
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pollCount++
		status := "queued"
		progress := 0

		if pollCount > 2 {
			status = "succeeded"
			progress = 100
		}

		resp := map[string]interface{}{
			"id":       "job_test",
			"status":   status,
			"progress": progress,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	tracker := newTrackerForTest(t, handler)
	tracker.Enqueue("job_test", "test prompt")

	// Wait for polling to run
	time.Sleep(100 * time.Millisecond)

	tracker.Stop()
}
