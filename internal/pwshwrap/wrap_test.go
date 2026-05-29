package pwshwrap

import (
	"strings"
	"testing"
)

func TestBuild_BasicCommand(t *testing.T) {
	spec, found := Build("npm run dev", 3000, "C:\\proj", nil, nil, nil)

	if found {
		t.Error("no {PORT} placeholder, found should be false")
	}
	if len(spec.Cmdline) == 0 || spec.Cmdline[0] != "powershell.exe" {
		t.Error("expected powershell.exe as first argv element")
	}
	if spec.Cwd != "C:\\proj" {
		t.Errorf("unexpected cwd: %s", spec.Cwd)
	}
	if spec.Env["PORT"] != "3000" {
		t.Errorf("expected PORT=3000, got %s", spec.Env["PORT"])
	}
}

func TestBuild_WithPortPlaceholder(t *testing.T) {
	spec, found := Build("node server.js --port {PORT}", 8080, "C:\\app", nil, nil, nil)

	if !found {
		t.Error("{PORT} present, found should be true")
	}
	// Last element of argv should contain "8080".
	last := spec.Cmdline[len(spec.Cmdline)-1]
	if !strings.Contains(last, "8080") {
		t.Errorf("argv[-1] should contain port: %q", last)
	}
}

func TestBuild_ExtraEnvMerged(t *testing.T) {
	extra := map[string]string{"FOO": "bar"}
	spec, _ := Build("cmd", 4000, "C:\\", extra, nil, nil)

	if spec.Env["FOO"] != "bar" {
		t.Error("extra env not merged")
	}
	if spec.Env["PORT"] != "4000" {
		t.Errorf("PORT not set: got %s", spec.Env["PORT"])
	}
}

func TestMergeEnv_OverridesExisting(t *testing.T) {
	env := mergeEnv(map[string]string{"PORT": "9999"}, 1234)
	// explicit extra overrides; then PORT is overwritten by injected port
	if env["PORT"] != "1234" {
		t.Errorf("injected PORT should win, got %s", env["PORT"])
	}
}
