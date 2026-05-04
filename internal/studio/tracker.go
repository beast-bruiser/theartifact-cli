package studio

import (
	"fmt"
	"sync"
	"time"

	"theartifact-cli/internal/api"
	"theartifact-cli/internal/ui"
)

// TrackedJob represents a single generation job being monitored.
type TrackedJob struct {
	ID       string
	ShortID  string
	Prompt   string
	Status   string // queued, processing, complete, failed
	Progress int
	Result   *api.JobResult
	Error    string
	QueuedAt time.Time
}

// JobTracker manages a pool of background jobs and polls their status.
type JobTracker struct {
	client     *api.Client
	mu         sync.Mutex
	jobs       []*TrackedJob
	onComplete func(*TrackedJob) // called from the poller goroutine
	stopCh     chan struct{}
}

// NewJobTracker creates a tracker and starts the background polling goroutine.
func NewJobTracker(client *api.Client, onComplete func(*TrackedJob)) *JobTracker {
	t := &JobTracker{
		client:     client,
		onComplete: onComplete,
		stopCh:     make(chan struct{}),
	}
	go t.pollLoop()
	return t
}

// Enqueue adds a job to be tracked.
func (t *JobTracker) Enqueue(jobID, prompt string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.jobs = append(t.jobs, &TrackedJob{
		ID:       jobID,
		ShortID:  shortID(jobID),
		Prompt:   prompt,
		Status:   "queued",
		QueuedAt: time.Now(),
	})
}

// Snapshot returns a copy of the current job list (safe to read outside lock).
func (t *JobTracker) Snapshot() []TrackedJob {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]TrackedJob, len(t.jobs))
	for i, j := range t.jobs {
		out[i] = *j
	}
	return out
}

// PendingCount returns the number of jobs not yet in a terminal state.
func (t *JobTracker) PendingCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	count := 0
	for _, j := range t.jobs {
		if j.Status == "queued" || j.Status == "running" {
			count++
		}
	}
	return count
}

// Stop halts the polling goroutine.
func (t *JobTracker) Stop() {
	close(t.stopCh)
}

// pollLoop runs in the background, polling all non-terminal jobs every 3 seconds.
func (t *JobTracker) pollLoop() {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.pollOnce()
		}
	}
}

func (t *JobTracker) pollOnce() {
	t.mu.Lock()
	// Gather IDs of pending jobs under lock
	pending := make([]*TrackedJob, 0)
	for _, j := range t.jobs {
		if j.Status == "queued" || j.Status == "running" {
			pending = append(pending, j)
		}
	}
	t.mu.Unlock()

	for _, job := range pending {
		updated, err := t.client.GetJob(job.ID)
		if err != nil {
			// Transient error — skip this cycle
			continue
		}

		t.mu.Lock()
		job.Status = updated.Status
		job.Progress = updated.Progress
		job.Result = updated.Result
		job.Error = updated.Error
		terminal := updated.Status == "succeeded" || updated.Status == "failed"
		t.mu.Unlock()

		if terminal {
			// Fire callback outside the lock so it can safely print
			t.onComplete(job)
		}
	}
}

// shortID returns the first 8 characters of a UUID for display.
func shortID(id string) string {
	if len(id) >= 8 {
		return id[:8]
	}
	return id
}

// PrintJobsTable renders the current job list as a styled table.
func (t *JobTracker) PrintJobsTable() {
	snap := t.Snapshot()
	if len(snap) == 0 {
		fmt.Println(ui.MutedStyle.Render("  No jobs queued yet."))
		return
	}

	headers := []string{"Job ID", "Status", "Progress", "Prompt"}
	rows := make([][]string, len(snap))
	for i, j := range snap {
		progress := fmt.Sprintf("%d%%", j.Progress)
		if j.Status == "succeeded" {
			progress = "100%"
		}
		prompt := j.Prompt
		if len(prompt) > 40 {
			prompt = prompt[:37] + "..."
		}
		rows[i] = []string{
			j.ShortID,
			ui.StatusStyle(j.Status).Render(j.Status),
			progress,
			prompt,
		}
	}
	ui.PrintTable(headers, rows)
}
