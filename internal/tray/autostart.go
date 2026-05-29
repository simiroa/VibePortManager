//ff:what Windows 자동시작: HKCU\Software\Microsoft\Windows\CurrentVersion\Run 레지스트리
//ff:why specs/manifest: autostart_vpm default=false; 사용자가 설정창에서 토글
package tray

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const (
	runKey        = `Software\Microsoft\Windows\CurrentVersion\Run`
	appName       = "VibePortManager"
	daemonAppName = "VibePortManagerDaemon"
)

// SetAutostart enables or disables Windows startup registry entry.
func SetAutostart(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry run key: %w", err)
	}
	defer k.Close()

	if !enable {
		return k.DeleteValue(appName)
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	return k.SetStringValue(appName, `"`+exe+`"`)
}

// SetDaemonAutostart enables/disables running the headless daemon (`vpm --daemon`)
// at login. Registered under a distinct Run-key value so it is independent of the
// GUI autostart entry.
func SetDaemonAutostart(enable bool) error {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("open registry run key: %w", err)
	}
	defer k.Close()

	if !enable {
		return k.DeleteValue(daemonAppName)
	}
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("get executable path: %w", err)
	}
	return k.SetStringValue(daemonAppName, `"`+exe+`" --daemon`)
}

// IsAutostartEnabled checks whether the registry entry exists.
func IsAutostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(appName)
	return err == nil
}
