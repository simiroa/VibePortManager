package wslwrap

import "testing"

func TestParseDistroList(t *testing.T) {
	input := "  NAME            STATE           VERSION\r\n* Ubuntu          Running         2\r\n  Debian          Stopped         2\r\n"
	distros := parseDistroList(input)

	if len(distros) != 2 {
		t.Fatalf("expected 2 distros, got %d", len(distros))
	}
	if distros[0].Name != "Ubuntu" || !distros[0].Default || distros[0].State != "Running" {
		t.Errorf("unexpected Ubuntu: %+v", distros[0])
	}
	if distros[1].Name != "Debian" || distros[1].Default || distros[1].State != "Stopped" {
		t.Errorf("unexpected Debian: %+v", distros[1])
	}
}

func TestParseDistroList_Empty(t *testing.T) {
	distros := parseDistroList("  NAME  STATE  VERSION\r\n")
	if len(distros) != 0 {
		t.Errorf("expected 0 distros, got %d", len(distros))
	}
}

func TestWinPathToWSL(t *testing.T) {
	cases := []struct {
		path, distro, want string
	}{
		{`\\wsl$\Ubuntu\home\user\app`, "Ubuntu", "/home/user/app"},
		{`\\wsl.localhost\Ubuntu\etc\hosts`, "Ubuntu", "/etc/hosts"},
		{`C:\Users\me\proj`, "Ubuntu", `C:\Users\me\proj`}, // fallback
	}
	for _, c := range cases {
		got := winPathToWSL(c.path, c.distro)
		if got != c.want {
			t.Errorf("winPathToWSL(%q, %q) = %q, want %q", c.path, c.distro, got, c.want)
		}
	}
}

func TestEscapeBash(t *testing.T) {
	got := escapeBash(`echo "hello $world"`)
	want := `echo \"hello $world\"`
	if got != want {
		t.Errorf("escapeBash = %q, want %q", got, want)
	}
}
