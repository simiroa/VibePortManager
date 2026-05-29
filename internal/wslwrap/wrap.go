//ff:what WSL distro 안의 bash -ilc 호출 argv 조립
//ff:why .bashrc 기반 NVM/FNM init이 interactive+login에서만 동작
package wslwrap

import (
	"fmt"
	"strings"

	vmsys "github.com/user/vpm/pkg/syscall"
)

// Build constructs a SpawnSpec that runs cmd inside the given WSL distro.
// Result argv: wsl.exe -d <distro> -- bash -ilc "<substituted cmd>"
// PORT is injected via WSLENV so the distro process can see it as an env var.
func Build(distro, cmd string, port int, cwd string, extraEnv map[string]string, stdout, stderr interface{ Write([]byte) (int, error) }) (vmsys.SpawnSpec, bool) {
	substituted, found := substitute(cmd, port)
	// Escape double-quotes for bash -c argument.
	inner := escapeBash(substituted)

	argv := []string{
		"bash", "-ilc", inner,
	}

	env := buildWSLEnv(extraEnv, port)

	return vmsys.SpawnSpec{
		Cmdline:   argv,
		Cwd:       winPathToWSL(cwd, distro),
		Env:       env,
		Stdout:    stdout,
		Stderr:    stderr,
		NewPGroup: true,
	}, found
}

// winPathToWSL converts a Windows UNC path (\\wsl$\Ubuntu\...) to /... for use as WSL cwd.
// Falls back to $HOME if conversion fails.
func winPathToWSL(path, distro string) string {
	prefix := `\\wsl$\` + distro
	prefixAlt := `\\wsl.localhost\` + distro
	if strings.HasPrefix(path, prefix) {
		rest := strings.TrimLeft(path[len(prefix):], `\`)
		return "/" + strings.ReplaceAll(rest, `\`, "/")
	}
	if strings.HasPrefix(path, prefixAlt) {
		rest := strings.TrimLeft(path[len(prefixAlt):], `\`)
		return "/" + strings.ReplaceAll(rest, `\`, "/")
	}
	return path
}

// buildWSLEnv creates environment with WSLENV to forward PORT into the distro.
func buildWSLEnv(extra map[string]string, port int) map[string]string {
	env := map[string]string{
		"PORT":   fmt.Sprintf("%d", port),
		"WSLENV": "PORT/u", // forward PORT as Unix var
	}
	for k, v := range extra {
		env[k] = v
	}
	return env
}

func substitute(cmd string, port int) (string, bool) {
	token := "{PORT}"
	portStr := fmt.Sprintf("%d", port)
	if strings.Contains(cmd, token) {
		return strings.ReplaceAll(cmd, token, portStr), true
	}
	return cmd, false
}

func escapeBash(s string) string {
	// Escape " and $ to prevent shell injection in bash -c "..."
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return s
}
