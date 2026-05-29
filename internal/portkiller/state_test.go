package portkiller

import "testing"

func TestState_String(t *testing.T) {
	cases := []struct {
		s    State
		want string
	}{
		{GracefulSent, "GracefulSent"},
		{Polling, "Polling"},
		{Released, "Released"},
		{ResolveBlocker, "ResolveBlocker"},
		{ForceKill, "ForceKill"},
		{CrossTargetReport, "CrossTargetReport"},
		{UnknownBlocker, "UnknownBlocker"},
		{State(99), "Unknown"},
	}
	for _, c := range cases {
		if got := c.s.String(); got != c.want {
			t.Errorf("State(%d).String() = %q, want %q", int(c.s), got, c.want)
		}
	}
}
