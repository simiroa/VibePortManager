//ff:what 로그 파일 로테이션: 7일 + 프로젝트당 100MB 상한
//ff:why specs/manifest.yaml: log.retention_days=7, per_project_cap_mb=100
package logbuf

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	retentionDays  = 7
	perProjectCapB = 100 * 1024 * 1024 // 100 MB
)

// LogDir returns the directory for a project's log files.
// Base: %APPDATA%/vpm/logs/<projectID>/
func LogDir(projectID string) (string, error) {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return "", fmt.Errorf("APPDATA env not set")
	}
	dir := filepath.Join(appdata, "vpm", "logs", projectID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// NewLogFile creates a new timestamped log file for a server run.
// Format: {YYYY-MM-DD}_{HHMMSS}_{serverID}.log  (UTC)
func NewLogFile(projectID, serverID string) (*os.File, string, error) {
	dir, err := LogDir(projectID)
	if err != nil {
		return nil, "", err
	}
	now := time.Now().UTC()
	name := fmt.Sprintf("%s_%s_%s.log",
		now.Format("2006-01-02"),
		now.Format("150405"),
		serverID,
	)
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		return nil, "", err
	}
	return f, path, nil
}

// Rotate deletes logs in dir older than 7 days and trims total size to 100 MB.
// Deletes oldest files first when over cap.
func Rotate(projectID string) error {
	dir, err := LogDir(projectID)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type fileInfo struct {
		path    string
		modTime time.Time
		size    int64
	}

	var files []fileInfo
	cutoff := time.Now().UTC().AddDate(0, 0, -retentionDays)

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dir, e.Name())
		if info.ModTime().Before(cutoff) {
			os.Remove(path)
			continue
		}
		files = append(files, fileInfo{path, info.ModTime(), info.Size()})
	}

	// Sort oldest first for cap enforcement.
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.Before(files[j].modTime)
	})

	var total int64
	for _, f := range files {
		total += f.size
	}
	for total > perProjectCapB && len(files) > 0 {
		os.Remove(files[0].path)
		total -= files[0].size
		files = files[1:]
	}
	return nil
}
