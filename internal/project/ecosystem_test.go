package project

import (
	"os"
	"path/filepath"
	"testing"
)

const sampleEcosystem = `module.exports = {
  apps: [
    {
      name: 'banana-frontend',
      script: 'C:\\Program Files\\nodejs\\node.exe',
      args: '"C:\\Program Files\\nodejs\\node_modules\\npm\\bin\\npm-cli.js" run dev -- --host 0.0.0.0 --port 5183 --strictPort',
      cwd: './',
      env: { NODE_ENV: 'development' }
    },
    {
      name: 'banana-backend',
      script: './banana-api/.venv/Scripts/uvicorn.exe',
      args: 'app.main:app --host 0.0.0.0 --port 8010',
      cwd: './banana-api',
      env: { PYTHONPATH: '.', BANANA_ENV: 'dev' }
    },
    {
      name: 'pm2-webui',
      script: 'C:\\Program Files\\nodejs\\node.exe',
      args: '"C:\\Program Files\\nodejs\\node_modules\\npm\\bin\\npm-cli.js" start',
      cwd: './tools/pm2-webui',
      env: { PORT: 4343, HOST: '0.0.0.0' }
    }
  ]
};`

func writeEcosystem(t *testing.T, name, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestDetectEcosystem(t *testing.T) {
	dir := writeEcosystem(t, "ecosystem.config.cjs", sampleEcosystem)
	got := DetectEcosystem(dir)

	if len(got) != 3 {
		t.Fatalf("expected 3 servers, got %d: %+v", len(got), got)
	}

	byName := map[string]DetectedServer{}
	for _, s := range got {
		byName[s.Name] = s
	}

	fe := byName["banana-frontend"]
	if fe.Port != 5183 {
		t.Errorf("frontend port = %d, want 5183", fe.Port)
	}
	if fe.Command != "npm run dev -- --host 0.0.0.0 --port 5183 --strictPort" {
		t.Errorf("frontend command = %q", fe.Command)
	}

	be := byName["banana-backend"]
	if be.Port != 8010 {
		t.Errorf("backend port = %d, want 8010", be.Port)
	}
	if be.Command != `cd /d banana-api && .venv\Scripts\uvicorn.exe app.main:app --host 0.0.0.0 --port 8010` {
		t.Errorf("backend command = %q", be.Command)
	}

	wu := byName["pm2-webui"]
	if wu.Port != 4343 {
		t.Errorf("webui port = %d, want 4343 (from env PORT)", wu.Port)
	}
	if wu.Command != `cd /d tools\pm2-webui && npm start` {
		t.Errorf("webui command = %q", wu.Command)
	}
}

func TestDetectEcosystemNone(t *testing.T) {
	if got := DetectEcosystem(t.TempDir()); got != nil {
		t.Errorf("expected nil for dir without ecosystem config, got %+v", got)
	}
}
