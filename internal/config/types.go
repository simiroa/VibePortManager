//ff:what config.schema.json → Go struct (gen-types.go 산출물 수동 적용)
//ff:why SSOT: JSON Schema가 정의한 형태와 1:1 매핑
package config

// Config is the top-level %APPDATA%/vpm/config.json structure.
// schema-hash: vpm.config.v1 (config.schema.json)
type Config struct {
	Version  int       `json:"version"`
	Settings Settings  `json:"settings"`
	Projects []Project `json:"projects"`
}

type Settings struct {
	AutostartVPM      bool                `json:"autostart_vpm"`
	CloseWarningSeen  bool                `json:"close_warning_seen"`
	ShellOverride     map[string][]string `json:"shell_override,omitempty"`
}

type Project struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Path            string   `json:"path"`
	ExecutionTarget string   `json:"execution_target"` // "windows-native"|"wsl"|"linux-native"
	WSLDistro       string   `json:"wsl_distro,omitempty"`
	PackageManager  string   `json:"package_manager"` // "npm"|"yarn"|"pnpm"|"bun"|"none"
	Servers         []Server `json:"servers"`
}

type Server struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Command     string `json:"command"`
	Port        int    `json:"port"`
	Autostart   bool   `json:"autostart"`
	Autorestart bool   `json:"autorestart,omitempty"` // relaunch on unexpected (crash) exit
}
