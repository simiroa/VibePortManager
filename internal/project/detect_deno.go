//ff:what Deno 탐지 (deno.json/deno.jsonc tasks의 dev/start/serve → deno task)
//ff:why Deno 프로젝트 범용 대응. 포트 미상(사용자 입력)
package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
)

type denoDetector struct{}

func (denoDetector) Name() string { return "deno" }

// reLineComment strips // line comments so deno.jsonc parses as JSON.
var reLineComment = regexp.MustCompile(`(?m)^\s*//.*$`)

func (denoDetector) Detect(dir string) []DetectedServer {
	for _, f := range []string{"deno.json", "deno.jsonc"} {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			continue
		}
		var doc struct {
			Tasks map[string]string `json:"tasks"`
		}
		if json.Unmarshal(reLineComment.ReplaceAll(data, []byte{}), &doc) != nil {
			return nil
		}
		for _, t := range []string{"dev", "start", "serve"} {
			if _, ok := doc.Tasks[t]; ok {
				return []DetectedServer{{
					Name:    filepath.Base(dir) + "-deno",
					Command: "deno task " + t,
					Port:    0,
					Source:  "deno",
				}}
			}
		}
		return nil
	}
	return nil
}
