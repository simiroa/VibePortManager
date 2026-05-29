//ff:what 설치된 WSL distro 목록
//ff:why Tab 1 "WSL Project 추가" 드롭다운, execution_target 자동 감지
package wslwrap

import (
	"fmt"
	"os/exec"
	"strings"

	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Distro represents an installed WSL distribution.
type Distro struct {
	Name    string
	Default bool
	State   string // "Running" | "Stopped"
}

// List returns all installed WSL distros.
// wsl.exe --list --quiet outputs UTF-16 LE text.
func List() ([]Distro, error) {
	out, err := exec.Command("wsl.exe", "--list", "--verbose").Output()
	if err != nil {
		return nil, fmt.Errorf("wsl --list --verbose: %w", err)
	}
	decoded, err := decodeUTF16(out)
	if err != nil {
		return nil, fmt.Errorf("utf16 decode: %w", err)
	}
	return parseDistroList(decoded), nil
}

// decodeUTF16 decodes a UTF-16 LE byte slice (with optional BOM) to a UTF-8 string.
func decodeUTF16(b []byte) (string, error) {
	dec := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewDecoder()
	utf8, _, err := transform.Bytes(dec, b)
	if err != nil {
		// Fallback: already UTF-8 (some wsl.exe versions output UTF-8).
		return string(b), nil
	}
	return string(utf8), nil
}

// parseDistroList parses the --verbose output:
//   NAME            STATE           VERSION
// * Ubuntu          Running         2
//   Debian          Stopped         2
func parseDistroList(text string) []Distro {
	var distros []Distro
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimRight(line, "\r")
		isDefault := strings.HasPrefix(line, "*")
		line = strings.TrimLeft(line, "* \t")
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[0]
		if name == "NAME" {
			continue // header row
		}
		state := "Stopped"
		if len(fields) >= 2 {
			state = fields[1]
		}
		distros = append(distros, Distro{Name: name, Default: isDefault, State: state})
	}
	return distros
}
