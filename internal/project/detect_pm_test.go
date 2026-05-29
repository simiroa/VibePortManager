package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/user/vpm/internal/project"
)

func TestDetectPackageManager(t *testing.T) {
	cases := []struct {
		file string
		want string
	}{
		{"bun.lockb", "bun"},
		{"pnpm-lock.yaml", "pnpm"},
		{"yarn.lock", "yarn"},
		{"package-lock.json", "npm"},
	}

	for _, c := range cases {
		t.Run(c.want, func(t *testing.T) {
			dir := t.TempDir()
			os.WriteFile(filepath.Join(dir, c.file), []byte{}, 0644)
			got := project.DetectPackageManager(dir)
			if got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}

	t.Run("none", func(t *testing.T) {
		dir := t.TempDir()
		got := project.DetectPackageManager(dir)
		if got != "none" {
			t.Errorf("got %q, want none", got)
		}
	})
}
