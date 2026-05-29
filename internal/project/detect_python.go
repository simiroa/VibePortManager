//ff:what Python 웹 프레임워크 탐지 (Django / FastAPI·uvicorn / Flask)
//ff:why 백엔드가 Python인 프로젝트도 범용 대응 (예: Bananadancer banana-api)
package project

import (
	"path/filepath"
	"strings"
)

type pythonDetector struct{}

func (pythonDetector) Name() string { return "python" }

func (pythonDetector) Detect(dir string) []DetectedServer {
	base := filepath.Base(dir)

	// Django — manage.py is unambiguous; runserver default port 8000.
	if exists(filepath.Join(dir, "manage.py")) {
		cmd := venvBin(dir, "python") + " manage.py runserver 0.0.0.0:8000"
		return []DetectedServer{{Name: base + "-django", Command: cmd, Port: 8000, Source: "python"}}
	}

	deps := readLower(dir, "requirements.txt") + readLower(dir, "pyproject.toml") + readLower(dir, "Pipfile")

	// FastAPI / Starlette via uvicorn.
	if strings.Contains(deps, "uvicorn") || strings.Contains(deps, "fastapi") {
		if mod := uvicornModule(dir); mod != "" {
			cmd := venvBin(dir, "uvicorn") + " " + mod + " --host 0.0.0.0 --port 8000 --reload"
			return []DetectedServer{{Name: base + "-api", Command: cmd, Port: 8000, Source: "python"}}
		}
	}

	// Flask.
	if strings.Contains(deps, "flask") {
		if app := firstExisting(dir, "app.py", "wsgi.py", "main.py", "server.py"); app != "" {
			cmd := venvBin(dir, "flask") + " --app " + strings.TrimSuffix(app, ".py") + " run --host 0.0.0.0 --port 5000"
			return []DetectedServer{{Name: base + "-flask", Command: cmd, Port: 5000, Source: "python"}}
		}
	}
	return nil
}

// uvicornModule guesses the ASGI app path (module:app) from common layouts.
func uvicornModule(dir string) string {
	switch {
	case exists(filepath.Join(dir, "app", "main.py")):
		return "app.main:app"
	case exists(filepath.Join(dir, "src", "main.py")):
		return "src.main:app"
	case exists(filepath.Join(dir, "main.py")):
		return "main:app"
	case exists(filepath.Join(dir, "app.py")):
		return "app:app"
	}
	return ""
}

func firstExisting(dir string, names ...string) string {
	for _, n := range names {
		if exists(filepath.Join(dir, n)) {
			return n
		}
	}
	return ""
}
