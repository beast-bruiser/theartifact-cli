package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScaffoldProject(t *testing.T) {
	dir := t.TempDir()

	if err := ScaffoldProject(dir); err != nil {
		t.Fatalf("ScaffoldProject: %v", err)
	}

	for _, sub := range []string{".artifact", ".artifact/brain", ".artifact/input", ".artifact/assets"} {
		if fi, err := os.Stat(filepath.Join(dir, sub)); err != nil || !fi.IsDir() {
			t.Errorf("expected directory %s to exist", sub)
		}
	}

	readmePath := filepath.Join(dir, ".artifact", "brain", ".README")
	if _, err := os.Stat(readmePath); err != nil {
		t.Errorf("expected brain/.README to exist: %v", err)
	}
}

func TestScaffoldProject_Idempotent(t *testing.T) {
	dir := t.TempDir()

	if err := ScaffoldProject(dir); err != nil {
		t.Fatalf("first scaffold: %v", err)
	}
	if err := ScaffoldProject(dir); err != nil {
		t.Fatalf("second scaffold: %v", err)
	}
}

func TestProjectConfig_SaveLoad(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	pc := &ProjectConfig{
		WorkspaceID:   "ws_test123",
		WorkspaceName: "Test Workspace",
	}

	if err := SaveProject(pc); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	if !ProjectInitialized() {
		t.Fatal("ProjectInitialized should return true after SaveProject")
	}

	loaded, err := LoadProject()
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if loaded.WorkspaceID != pc.WorkspaceID {
		t.Errorf("workspace_id: got %q, want %q", loaded.WorkspaceID, pc.WorkspaceID)
	}
	if loaded.WorkspaceName != pc.WorkspaceName {
		t.Errorf("workspace_name: got %q, want %q", loaded.WorkspaceName, pc.WorkspaceName)
	}
}

func TestProjectInitialized_False(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	if ProjectInitialized() {
		t.Fatal("ProjectInitialized should return false in empty dir")
	}
}
