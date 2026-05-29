package pwshwrap

import "testing"

func TestSubstitute_Found(t *testing.T) {
	out, found := Substitute("npm run dev -- --port {PORT}", 3000)
	if !found {
		t.Error("expected placeholder found")
	}
	if out != "npm run dev -- --port 3000" {
		t.Errorf("unexpected output: %q", out)
	}
}

func TestSubstitute_NotFound(t *testing.T) {
	out, found := Substitute("npm start", 3000)
	if found {
		t.Error("expected placeholder not found")
	}
	if out != "npm start" {
		t.Errorf("cmd should be unchanged, got %q", out)
	}
}

func TestSubstitute_MultipleOccurrences(t *testing.T) {
	out, found := Substitute("cmd {PORT} --also {PORT}", 8080)
	if !found {
		t.Error("expected found")
	}
	if out != "cmd 8080 --also 8080" {
		t.Errorf("got %q", out)
	}
}

func TestEscapeForCmd(t *testing.T) {
	cases := []struct{ in, want string }{
		{`hello "world"`, `hello \"world\"`},
		{`no quotes`, `no quotes`},
	}
	for _, c := range cases {
		got := escapeForCmd(c.in)
		if got != c.want {
			t.Errorf("escapeForCmd(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestItoa(t *testing.T) {
	cases := []struct {
		n    int
		want string
	}{
		{0, "0"},
		{1, "1"},
		{3000, "3000"},
		{-5, "-5"},
		{65535, "65535"},
	}
	for _, c := range cases {
		if got := itoa(c.n); got != c.want {
			t.Errorf("itoa(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}
