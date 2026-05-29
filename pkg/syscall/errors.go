//ff:what sentinel errors for OS abstraction layer
//ff:why callers switch on these; no string comparison
package syscall

import "errors"

var (
	ErrNotFound    = errors.New("process not found")
	ErrPermission  = errors.New("permission denied")
	ErrCrossTarget = errors.New("process belongs to a different execution target")
	ErrBusy        = errors.New("operation already in progress")
	ErrDistroDown  = errors.New("WSL distro is not running")
)
