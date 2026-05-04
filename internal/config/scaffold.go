package config

import (
	"os"
	"path/filepath"
)

const brainReadme = `# brain/

This directory is owned by TheArtifact server.
Ingest and critique jobs overwrite the files listed in their results.
Do not hand-edit these files — your changes will be overwritten.
`

// ScaffoldProject creates the standard project layout inside .artifact/:
//
//	.artifact/
//	  config.json  — project config (workspace binding)
//	  brain/       — server-managed knowledge files
//	  input/       — files to upload for ingest
//	  assets/      — downloaded generation outputs
func ScaffoldProject(dir string) error {
	base := filepath.Join(dir, ".artifact")
	dirs := []string{
		base,
		filepath.Join(base, "brain"),
		filepath.Join(base, "input"),
		filepath.Join(base, "assets"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}

	readmePath := filepath.Join(base, "brain", ".README")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		if err := os.WriteFile(readmePath, []byte(brainReadme), 0644); err != nil {
			return err
		}
	}

	return nil
}
