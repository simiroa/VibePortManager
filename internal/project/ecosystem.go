//ff:what PM2 ecosystem.config.{cjs,js,json}에서 다중 서버 후보 추출
//ff:why PM2 런타임 의존 없이 설정 "포맷"만 읽어 VPM 서버로 import (AGPL 무관, 데이터 파싱)
package project

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// ecosystemDetector imports servers from a PM2 ecosystem config.
type ecosystemDetector struct{}

func (ecosystemDetector) Name() string                    { return "pm2" }
func (ecosystemDetector) Detect(dir string) []DetectedServer { return DetectEcosystem(dir) }

var (
	reAppName = regexp.MustCompile(`name\s*:\s*['"]([^'"]+)['"]`)
	reCwd     = quotePair("cwd")
	reScript  = quotePair("script")
	reArgs    = quotePair("args")
	rePortArg = regexp.MustCompile(`--port[=\s]+(\d{2,5})`)
	rePortEnv = regexp.MustCompile(`(?i)\bport\b\s*:\s*['"]?(\d{2,5})`)
)

// qpair matches a `key: '...'` or `key: "..."` field. Two patterns (not one
// with a char class) so a value can safely contain the *other* quote char —
// PM2 args are commonly single-quoted but embed double-quoted paths.
type qpair struct{ sq, dq *regexp.Regexp }

func quotePair(key string) qpair {
	return qpair{
		sq: regexp.MustCompile(key + `\s*:\s*'([^']*)'`),
		dq: regexp.MustCompile(key + `\s*:\s*"([^"]*)"`),
	}
}

func (q qpair) find(s string) string {
	if m := q.sq.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	if m := q.dq.FindStringSubmatch(s); m != nil {
		return m[1]
	}
	return ""
}

// DetectEcosystem parses a PM2 ecosystem config in dir (if present) and returns
// one DetectedServer per app that has a resolvable port. We never execute or
// link PM2 — this only reads the declarative config file as data.
func DetectEcosystem(dir string) []DetectedServer {
	var text string
	for _, name := range []string{"ecosystem.config.cjs", "ecosystem.config.js", "ecosystem.config.json"} {
		if b, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			text = string(b)
			break
		}
	}
	if text == "" {
		return nil
	}

	// Each app is delimited by its `name:` field; slice from one name to the next.
	locs := reAppName.FindAllStringSubmatchIndex(text, -1)
	if len(locs) == 0 {
		return nil
	}

	out := make([]DetectedServer, 0, len(locs))
	for i, loc := range locs {
		end := len(text)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		block := text[loc[0]:end]
		ds := DetectedServer{
			Name:    text[loc[2]:loc[3]],
			Port:    parsePort(block),
			Command: buildCommand(block),
			Source:  "pm2",
		}
		if ds.Port > 0 && ds.Command != "" {
			out = append(out, ds)
		}
	}
	return out
}

func parsePort(block string) int {
	if m := rePortArg.FindStringSubmatch(block); m != nil {
		return atoi(m[1])
	}
	if m := rePortEnv.FindStringSubmatch(block); m != nil {
		return atoi(m[1])
	}
	return 0
}

// buildCommand reconstructs a best-effort runnable command from script/args/cwd.
// The result is shown in an editable proposal field, so rough is acceptable.
func buildCommand(block string) string {
	cwd := reCwd.find(block)
	script := reScript.find(block)
	args := reArgs.find(block)

	cwdNorm := strings.Trim(strings.TrimPrefix(cwd, "./"), `/\`)
	prefix := ""
	if cwdNorm != "" && cwdNorm != "." {
		prefix = "cd /d " + toWinPath(cwdNorm) + " && "
	}

	// node.exe + npm-cli.js indirection → clean "npm ..." form.
	low := strings.ToLower(script)
	if strings.Contains(low, "node") && strings.Contains(args, "npm-cli.js") {
		if idx := strings.Index(args, "npm-cli.js"); idx >= 0 {
			rest := strings.TrimLeft(args[idx+len("npm-cli.js"):], `"' `)
			return prefix + "npm " + strings.TrimSpace(rest)
		}
	}

	if script == "" {
		return ""
	}

	// Generic: relativise the script to cwd and quote if it contains spaces.
	s := toWinPath(stripCwd(script, cwd))
	if strings.Contains(s, " ") {
		s = `"` + s + `"`
	}
	cmd := s
	if args != "" {
		cmd += " " + args
	}
	return prefix + strings.TrimSpace(cmd)
}

func stripCwd(script, cwd string) string {
	s := strings.TrimPrefix(strings.ReplaceAll(script, "\\", "/"), "./")
	c := strings.Trim(strings.TrimPrefix(cwd, "./"), "/")
	if c != "" && strings.HasPrefix(s, c+"/") {
		return s[len(c)+1:]
	}
	return s
}

func toWinPath(p string) string { return strings.ReplaceAll(p, "/", `\`) }

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
