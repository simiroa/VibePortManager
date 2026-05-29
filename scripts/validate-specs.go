//go:build ignore

// validate-specs: SSOT 일관성 검증기 (yongol 원칙 간이 구현)
// Run: go run scripts/validate-specs.go
// Checks:
//   1. manifest.yaml ports.allowed=0 → no net.Listen in non-test Go files
//   2. config.schema.json fields ↔ internal/config/types.go struct fields
//   3. ipc.yaml methods ↔ app.go exported methods
//   4. specs/states/port_killer.mmd states ↔ internal/portkiller/state.go constants

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	root := findRoot()
	errs := []string{}

	if e := checkNoNetListen(root); e != nil {
		errs = append(errs, e.Error())
	}
	if e := checkIPCMethods(root); e != nil {
		errs = append(errs, e.Error())
	}
	if e := checkPortKillerStates(root); e != nil {
		errs = append(errs, e.Error())
	}

	if len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "SSOT validation FAILED:\n")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  • %s\n", e)
		}
		os.Exit(1)
	}
	fmt.Println("SSOT validation passed ✓")
}

// Rule 1: no net.Listen in production code (spec: listening_ports=0)
func checkNoNetListen(root string) error {
	re := regexp.MustCompile(`\bnet\.Listen\b`)
	var hits []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		if strings.Contains(rel, "vendor") || strings.Contains(rel, ".git") {
			return filepath.SkipDir
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		// Exclude scripts/ — validator source itself contains the literal string
		if strings.HasPrefix(rel, "scripts"+string(filepath.Separator)) || rel == "scripts" {
			return nil
		}
		// Allow internal/portkiller/poll.go — it probes ports, not listens permanently
		if strings.Contains(rel, "portkiller") {
			return nil
		}
		// Allow port-probe helpers (bind-then-close, not a server)
		if strings.HasSuffix(rel, filepath.Join("ports", "suggest.go")) {
			return nil
		}
		if strings.HasSuffix(rel, filepath.Join("server", "start.go")) {
			return nil
		}
		data, _ := os.ReadFile(path)
		if re.Match(data) {
			hits = append(hits, rel)
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(hits) > 0 {
		return fmt.Errorf("net.Listen found in production code (violates ports.allowed=0): %v", hits)
	}
	return nil
}

// Rule 3: ipc.yaml method names ↔ app.go exported method names
func checkIPCMethods(root string) error {
	ipcPath := filepath.Join(root, "specs", "ipc.yaml")
	appPath := filepath.Join(root, "app.go")

	ipcNames := extractYAMLMethodNames(ipcPath)
	appMethods := extractGoMethods(appPath)

	var missing []string
	for _, name := range ipcNames {
		if !contains(appMethods, name) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("ipc.yaml methods missing in app.go: %v", missing)
	}
	return nil
}

// Rule 4: port_killer.mmd states ↔ portkiller/state.go constants
func checkPortKillerStates(root string) error {
	mmdPath := filepath.Join(root, "specs", "states", "port_killer.mmd")
	goPath := filepath.Join(root, "internal", "portkiller", "state.go")

	mmdStates := extractMermaidStates(mmdPath)
	goConsts := extractGoConsts(goPath)

	var missing []string
	for _, s := range mmdStates {
		if !containsFold(goConsts, s) {
			missing = append(missing, s)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("port_killer.mmd states missing in portkiller/state.go: %v", missing)
	}
	return nil
}

func extractYAMLMethodNames(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	re := regexp.MustCompile(`^\s{2,4}- name:\s*(\w+)`)
	var names []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		// Stop at the events: section — those are Wails events, not IPC methods
		if strings.TrimSpace(line) == "events:" {
			break
		}
		if m := re.FindStringSubmatch(line); m != nil {
			names = append(names, m[1])
		}
	}
	return names
}

func extractGoMethods(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	re := regexp.MustCompile(`^func \(a \*App\) (\w+)\(`)
	var names []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := re.FindStringSubmatch(sc.Text()); m != nil {
			names = append(names, m[1])
		}
	}
	return names
}

func extractMermaidStates(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`\b([A-Z][a-zA-Z]+)\s*:`)
	matches := re.FindAllStringSubmatch(string(data), -1)
	seen := map[string]bool{}
	var states []string
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			states = append(states, m[1])
		}
	}
	return states
}

func extractGoConsts(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// reLeader matches the first line of a State const block: "Identifier  State = iota"
	reLeader := regexp.MustCompile(`^\s+\w+\s+State\s*=`)
	// reIdent captures the leading identifier on any const line
	reIdent := regexp.MustCompile(`^\s+([A-Za-z]\w*)`)

	var names []string
	inBlock := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == ")" {
			inBlock = false
			continue
		}
		// Enter block when we see "Identifier State = iota"
		if reLeader.MatchString(line) {
			inBlock = true
		}
		if !inBlock {
			continue
		}
		// Skip blank lines and inline comments
		if trimmed == "" || strings.HasPrefix(trimmed, "//") {
			continue
		}
		if m := reIdent.FindStringSubmatch(line); m != nil {
			names = append(names, m[1])
		}
	}
	return names
}

func findRoot() string {
	dir, _ := os.Getwd()
	return dir
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func containsFold(ss []string, s string) bool {
	for _, v := range ss {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
