//ff:what Windows 명령을 powershell.exe argv로 조립
//ff:why 사용자의 OS-level PATH/NVM 상속 보장
package pwshwrap

import (
	"os"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// Build constructs a SpawnSpec that runs cmd inside PowerShell on Windows.
// The user command is wrapped as: powershell.exe -NoProfile -ExecutionPolicy Bypass -Command "cmd /c <cmd>"
// PORT is always injected as an env var; if {PORT} placeholder is found it is also substituted.
func Build(cmd string, port int, cwd string, extraEnv map[string]string, stdout, stderr interface{ Write([]byte) (int, error) }) (vmsys.SpawnSpec, bool) {
	substituted, found := Substitute(cmd, port)
	inner := escapeForCmd(substituted)

	argv := []string{
		"powershell.exe",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-Command",
		`cmd /c "` + inner + `"`,
	}

	env := mergeEnv(extraEnv, port)

	return vmsys.SpawnSpec{
		Cmdline: argv,
		Cwd:     cwd,
		Env:     env,
		Stdout:  stdout,
		Stderr:  stderr,
	}, found
}

// mergeEnv merges host environment with extraEnv overrides and injects PORT.
func mergeEnv(extra map[string]string, port int) map[string]string {
	merged := make(map[string]string)
	for _, kv := range os.Environ() {
		for i, c := range kv {
			if c == '=' {
				merged[kv[:i]] = kv[i+1:]
				break
			}
		}
	}
	for k, v := range extra {
		merged[k] = v
	}
	merged["PORT"] = itoa(port)
	return merged
}
