//go:build windows

//ff:what WSL distro 내부 프로세스를 Windows 호스트에서 제어
//ff:why wsl.exe -d <distro> -- prefix로 모든 명령을 distro 안에서 실행
package syscall

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

type wslBackend struct {
	distro string
	mu     sync.Mutex
}

func newWSLBackend(distro string) Backend {
	return &wslBackend{distro: distro}
}

func (w *wslBackend) ID() string { return "wsl:" + w.distro }

func (w *wslBackend) Healthy() error {
	out, err := exec.Command("wsl.exe", "--list", "--running", "--quiet").Output()
	if err != nil {
		return fmt.Errorf("wsl --list: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.EqualFold(strings.TrimSpace(line), w.distro) {
			return nil
		}
	}
	return ErrDistroDown
}

func (w *wslBackend) Spawn(ctx context.Context, spec SpawnSpec) (Handle, error) {
	if len(spec.Cmdline) == 0 {
		return Handle{}, fmt.Errorf("empty cmdline")
	}
	// spec.Cmdline is already assembled as: ["bash", "-ilc", "<cmd>"]
	// We prepend "wsl.exe -d <distro> --"
	args := append([]string{"-d", w.distro, "--"}, spec.Cmdline...)
	cmd := exec.CommandContext(ctx, "wsl.exe", args...)
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	// NOTE: Cwd inside WSL must be a Linux path; the caller converts if needed.
	if spec.Cwd != "" {
		cmd.Dir = spec.Cwd
	}
	if err := cmd.Start(); err != nil {
		return Handle{}, fmt.Errorf("wsl spawn: %w", err)
	}
	// The PID here is the wsl.exe host process PID. The actual Linux PGID is
	// retrieved lazily when SignalTree is called.
	return Handle{PID: cmd.Process.Pid}, nil
}

func (w *wslBackend) SignalTree(handle Handle, sig Signal) error {
	sigNum := "TERM"
	if sig == SignalKill {
		sigNum = "KILL"
	}
	if handle.PGID != 0 {
		// Use negative PGID to target the whole process group.
		_, err := w.wslExec("kill", fmt.Sprintf("-%s", sigNum), fmt.Sprintf("-%d", handle.PGID))
		return err
	}
	_, err := w.wslExec("kill", fmt.Sprintf("-%s", sigNum), strconv.Itoa(handle.PID))
	return err
}

func (w *wslBackend) KillPID(pid int) error {
	_, err := w.wslExec("kill", "-9", strconv.Itoa(pid))
	return err
}

func (w *wslBackend) ResolvePortOwner(port int) (PortOwner, error) {
	// Try ss first, fall back to lsof.
	out, err := w.wslExec("ss", "-ltnp", fmt.Sprintf("sport = :%d", port))
	if err == nil && strings.Contains(out, "LISTEN") {
		pid := parseSSPID(out)
		if pid > 0 {
			desc := fmt.Sprintf("(PID %d, wsl:%s)", pid, w.distro)
			return PortOwner{PID: pid, Description: desc, Origin: OriginSameTarget}, nil
		}
	}
	// Fallback: lsof
	out, err = w.wslExec("lsof", "-t", "-i", fmt.Sprintf(":%d", port))
	if err == nil {
		pid, _ := strconv.Atoi(strings.TrimSpace(out))
		if pid > 0 {
			desc := fmt.Sprintf("(PID %d, wsl:%s)", pid, w.distro)
			return PortOwner{PID: pid, Description: desc, Origin: OriginSameTarget}, nil
		}
	}
	return PortOwner{}, nil
}

// ResolveTreePort is not supported for WSL (process-tree port re-detection is
// Windows-native only for now); returns 0 so callers treat it as "not found".
func (w *wslBackend) ResolveTreePort(_ int) (int, error) { return 0, nil }

// ResolveProcessCommand reads /proc/<pid>/cmdline inside the distro (NUL-separated
// args). Returns "" when unavailable — callers fall back to monitor-only.
func (w *wslBackend) ResolveProcessCommand(pid int) (string, error) {
	if pid <= 0 {
		return "", nil
	}
	out, err := w.wslExec("cat", fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return "", nil
	}
	cmd := strings.TrimSpace(strings.ReplaceAll(out, "\x00", " "))
	return cmd, nil
}

func (w *wslBackend) ScanListenPorts() ([]ListenEntry, error) {
	out, err := w.wslExec("ss", "-ltnp")
	if err != nil {
		return nil, fmt.Errorf("wsl ss: %w", err)
	}
	var entries []ListenEntry
	scanner := bufio.NewScanner(bytes.NewBufferString(out))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		// ss output: State Recv-Q Send-Q Local-Address:Port Peer-Address:Port Process
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		localField := fields[3]
		colon := strings.LastIndex(localField, ":")
		if colon < 0 {
			continue
		}
		portStr := localField[colon+1:]
		port, err := strconv.Atoi(portStr)
		if err != nil {
			continue
		}
		pid := parseSSPID(line)
		entries = append(entries, ListenEntry{
			Port:        port,
			PID:         pid,
			Description: fmt.Sprintf("(PID %d, wsl:%s)", pid, w.distro),
		})
	}
	return entries, nil
}

// wslExec runs a command inside the distro and returns combined output.
func (w *wslBackend) wslExec(args ...string) (string, error) {
	fullArgs := append([]string{"-d", w.distro, "--"}, args...)
	out, err := exec.Command("wsl.exe", fullArgs...).CombinedOutput()
	return string(out), err
}

// parseSSPID extracts the PID from ss -ltnp output.
// Output line example: LISTEN 0 128 0.0.0.0:3000 0.0.0.0:* users:(("node",pid=1234,fd=20))
func parseSSPID(data string) int {
	scanner := bufio.NewScanner(bytes.NewBufferString(data))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		idx := strings.Index(line, "pid=")
		if idx < 0 {
			continue
		}
		rest := line[idx+4:]
		end := strings.IndexAny(rest, ",)")
		if end < 0 {
			end = len(rest)
		}
		pid, err := strconv.Atoi(rest[:end])
		if err == nil {
			return pid
		}
	}
	return 0
}
