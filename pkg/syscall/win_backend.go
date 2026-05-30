//go:build windows

//ff:what Windows-native Executor: os/exec + taskkill + netstat
//ff:why Windows 프로세스 트리는 Job Object 또는 taskkill /T로 제어
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
	stdsyscall "syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

type winBackend struct {
	mu sync.Mutex
}

func newWinBackend() Backend {
	return &winBackend{}
}

func (w *winBackend) ID() string { return "windows-native" }

func (w *winBackend) Healthy() error { return nil }

func (w *winBackend) Spawn(ctx context.Context, spec SpawnSpec) (Handle, error) {
	if len(spec.Cmdline) == 0 {
		return Handle{}, fmt.Errorf("empty cmdline")
	}
	cmd := exec.CommandContext(ctx, spec.Cmdline[0], spec.Cmdline[1:]...)
	cmd.Dir = spec.Cwd
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr
	cmd.SysProcAttr = &stdsyscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: stdsyscall.CREATE_NEW_PROCESS_GROUP,
	}
	if len(spec.Env) > 0 {
		env := buildEnvSlice(spec.Env)
		cmd.Env = env
	}
	if err := cmd.Start(); err != nil {
		return Handle{}, fmt.Errorf("spawn: %w", err)
	}
	return Handle{PID: cmd.Process.Pid}, nil
}

// hiddenExec runs a command without a visible console window and with a
// hard 8-second timeout so a stalled netstat/tasklist never freezes the UI.
func hiddenExec(name string, args ...string) *exec.Cmd {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.SysProcAttr = &stdsyscall.SysProcAttr{HideWindow: true}
	// cancel is intentionally leaked: it fires at most 8 s after the call,
	// which is harmless for a long-running desktop process.
	_ = cancel
	return cmd
}

func (w *winBackend) SignalTree(handle Handle, sig Signal) error {
	args := []string{"/T", "/PID", strconv.Itoa(handle.PID)}
	if sig == SignalKill {
		args = append(args, "/F")
	}
	out, err := hiddenExec("taskkill", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("taskkill %v: %w (%s)", args, err, bytes.TrimSpace(out))
	}
	return nil
}

func (w *winBackend) KillPID(pid int) error {
	out, err := hiddenExec("taskkill", "/F", "/PID", strconv.Itoa(pid)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("taskkill /F /PID %d: %w (%s)", pid, err, bytes.TrimSpace(out))
	}
	return nil
}

func (w *winBackend) ResolvePortOwner(port int) (PortOwner, error) {
	out, err := hiddenExec("netstat", "-ano", "-p", "tcp").Output()
	if err != nil {
		return PortOwner{}, fmt.Errorf("netstat: %w", err)
	}
	pid := parseNetstatPID(out, port)
	if pid == 0 {
		return PortOwner{}, nil
	}
	desc := resolveProcessName(pid)
	return PortOwner{PID: pid, Description: desc, Origin: OriginSameTarget}, nil
}

// parseNetstatPID scans netstat -ano output for a LISTENING entry on port.
// Returns 0 if not found.
func parseNetstatPID(data []byte, port int) int {
	target := fmt.Sprintf(":%d", port)
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "LISTENING") {
			continue
		}
		fields := strings.Fields(line)
		// fields: Proto  LocalAddress  ForeignAddress  State  PID
		if len(fields) < 5 {
			continue
		}
		localAddr := fields[1]
		if !strings.HasSuffix(localAddr, target) {
			continue
		}
		pid, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil {
			continue
		}
		return pid
	}
	return 0
}

// resolveProcessName calls tasklist to get the executable name for a PID.
func resolveProcessName(pid int) string {
	out, err := hiddenExec("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
	if err != nil {
		return fmt.Sprintf("PID %d", pid)
	}
	// CSV: "image.exe","PID","Session Name","Session#","Mem Usage"
	line := strings.TrimSpace(string(out))
	if line == "" || strings.Contains(line, "No tasks") {
		return fmt.Sprintf("PID %d", pid)
	}
	parts := strings.Split(line, ",")
	if len(parts) > 0 {
		name := strings.Trim(parts[0], `"`)
		return fmt.Sprintf("%s (PID %d)", name, pid)
	}
	return fmt.Sprintf("PID %d", pid)
}

func (w *winBackend) ScanListenPorts() ([]ListenEntry, error) {
	entries, err := scanListenRaw()
	if err != nil {
		return nil, err
	}
	// Resolve names with a SINGLE tasklist call (pid→name map), not one tasklist
	// per port. On a machine with many listeners the per-port approach took tens
	// of seconds and made the UI look frozen.
	names := processNameMap()
	for i := range entries {
		if n, ok := names[entries[i].PID]; ok {
			entries[i].Description = fmt.Sprintf("%s (PID %d)", n, entries[i].PID)
		} else {
			entries[i].Description = fmt.Sprintf("PID %d", entries[i].PID)
		}
	}
	return entries, nil
}

// processNameMap returns pid→image-name for all processes via one tasklist call.
func processNameMap() map[int]string {
	out, err := hiddenExec("tasklist", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return nil
	}
	m := make(map[int]string)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		// CSV row: "image.exe","1234","Console","1","12,345 K"
		parts := strings.Split(strings.TrimSpace(scanner.Text()), `","`)
		if len(parts) < 2 {
			continue
		}
		pid, err := strconv.Atoi(strings.Trim(parts[1], `"`))
		if err != nil {
			continue
		}
		m[pid] = strings.Trim(parts[0], `"`)
	}
	return m
}

// scanListenRaw returns LISTEN port+PID pairs from a single netstat call,
// without per-PID name resolution. Cheap enough for repeated/internal use.
func scanListenRaw() ([]ListenEntry, error) {
	out, err := hiddenExec("netstat", "-ano", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("netstat: %w", err)
	}
	var entries []ListenEntry
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "LISTENING") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}
		localAddr := fields[1]
		pid, err := strconv.Atoi(fields[len(fields)-1])
		if err != nil {
			continue
		}
		colon := strings.LastIndex(localAddr, ":")
		if colon < 0 {
			continue
		}
		port, err := strconv.Atoi(localAddr[colon+1:])
		if err != nil {
			continue
		}
		entries = append(entries, ListenEntry{Port: port, PID: pid})
	}
	return entries, nil
}

// ResolveTreePort returns a LISTEN port owned by the process tree rooted at
// rootPID. The socket owner is usually a descendant (npm → cmd → node), so we
// build the descendant set via a process snapshot and intersect with netstat.
func (w *winBackend) ResolveTreePort(rootPID int) (int, error) {
	if rootPID <= 0 {
		return 0, nil
	}
	tree, err := descendantPIDs(rootPID)
	if err != nil {
		return 0, err
	}
	entries, err := scanListenRaw() // no per-port name resolution — keep it fast
	if err != nil {
		return 0, err
	}
	for _, e := range entries {
		if tree[e.PID] {
			return e.Port, nil
		}
	}
	return 0, nil
}

// ResolveProcessCommand returns the full command line for a PID via a CIM query.
// PowerShell's Get-CimInstance is reliable on Win10/11 (unlike the deprecated
// wmic). Returns "" (no error) when the PID is gone or the command line is empty.
func (w *winBackend) ResolveProcessCommand(pid int) (string, error) {
	if pid <= 0 {
		return "", nil
	}
	script := fmt.Sprintf("(Get-CimInstance Win32_Process -Filter \"ProcessId=%d\").CommandLine", pid)
	out, err := hiddenExec("powershell", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return "", nil // process may have exited or access denied — treat as unknown
	}
	return strings.TrimSpace(string(out)), nil
}

// descendantPIDs returns rootPID plus every descendant PID, via a Toolhelp
// process snapshot (one syscall, no external process).
func descendantPIDs(root int) (map[int]bool, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, fmt.Errorf("process snapshot: %w", err)
	}
	defer windows.CloseHandle(snap)

	children := map[int][]int{} // ppid -> child pids
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err := windows.Process32First(snap, &e); err != nil {
		return nil, fmt.Errorf("process32first: %w", err)
	}
	for {
		children[int(e.ParentProcessID)] = append(children[int(e.ParentProcessID)], int(e.ProcessID))
		if err := windows.Process32Next(snap, &e); err != nil {
			break // ERROR_NO_MORE_FILES ends the walk
		}
	}

	set := map[int]bool{root: true}
	queue := []int{root}
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		for _, c := range children[p] {
			if !set[c] {
				set[c] = true
				queue = append(queue, c)
			}
		}
	}
	return set, nil
}

func buildEnvSlice(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}
