//ff:what {PORT} 토큰 치환 + PORT 환경변수 fallback
//ff:why 프레임워크별 PORT 인식 차이 흡수 (Vite/Storybook은 CLI flag만 받음)
package pwshwrap

import "strings"

const portToken = "{PORT}"

// Substitute replaces {PORT} in cmd with the decimal string of port.
// Returns the substituted command and whether the placeholder was found.
// If not found, the caller should inject PORT as an env var and warn the user.
func Substitute(cmd string, port int) (substituted string, placeholderFound bool) {
	portStr := itoa(port)
	if strings.Contains(cmd, portToken) {
		return strings.ReplaceAll(cmd, portToken, portStr), true
	}
	return cmd, false
}

// escapeForCmd escapes double-quotes for use inside cmd /c "...".
func escapeForCmd(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}
