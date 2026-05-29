//ff:what docker-compose 서비스에서 호스트 포트가 매핑된 것들을 후보로 추출
//ff:why DB/백엔드 등 compose 기반 dev 환경도 범용 대응 (Docker 런타임 의존 없이 yml 파싱만)
package project

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type composeDetector struct{}

func (composeDetector) Name() string { return "compose" }

func (composeDetector) Detect(dir string) []DetectedServer {
	for _, name := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		if data, err := os.ReadFile(filepath.Join(dir, name)); err == nil {
			return parseCompose(data)
		}
	}
	return nil
}

func parseCompose(data []byte) []DetectedServer {
	var doc struct {
		Services map[string]struct {
			Ports []yaml.Node `yaml:"ports"`
		} `yaml:"services"`
	}
	if yaml.Unmarshal(data, &doc) != nil {
		return nil
	}
	out := make([]DetectedServer, 0, len(doc.Services))
	for svc, def := range doc.Services {
		port := firstHostPort(def.Ports)
		if port == 0 {
			continue // no published host port → internal-only service, skip
		}
		out = append(out, DetectedServer{
			Name:    svc,
			Command: "docker compose up " + svc,
			Port:    port,
			Source:  "compose",
		})
	}
	return out
}

// firstHostPort extracts the host-side port from compose `ports` entries:
// "8080:80", "127.0.0.1:8080:80", 8080, or { published: 8080, target: 80 }.
func firstHostPort(nodes []yaml.Node) int {
	for _, n := range nodes {
		switch n.Kind {
		case yaml.ScalarNode:
			parts := strings.Split(n.Value, ":")
			host := parts[0]
			if len(parts) == 3 { // ip:host:container
				host = parts[1]
			}
			if p, _ := strconv.Atoi(strings.TrimSpace(host)); p > 0 {
				return p
			}
		case yaml.MappingNode:
			for i := 0; i+1 < len(n.Content); i += 2 {
				if n.Content[i].Value == "published" {
					if p, _ := strconv.Atoi(n.Content[i+1].Value); p > 0 {
						return p
					}
				}
			}
		}
	}
	return 0
}
