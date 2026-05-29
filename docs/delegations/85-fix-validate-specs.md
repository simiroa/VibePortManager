# Delegation: Fix validate-specs.go (3 bugs)

## Context

`scripts/validate-specs.go` is the SSOT cross-layer consistency checker for VPM.
It must exit 0 (pass) before `docs/delegations/90-build-verify.md` can run.

Currently it reports 3 false-positive failures even though the code is correct.
All 3 are bugs in the validator itself, not in the code being validated.

Project root: `C:\Users\HG\Documents\Phase_yg`

## Verification Command

After fixing, run from project root:

```powershell
go run ./scripts/validate-specs.go
```

Expected output:

```
SSOT validation passed ✓
```

Exit code must be 0. Any other output = not done.

---

## Bug 1 — Rule 1: `net.Listen` false positive in `internal/ports/suggest.go`

### File

`scripts/validate-specs.go` — function `checkNoNetListen` (lines 47–78)

### Problem

`internal/ports/suggest.go` uses `net.Listen` to **probe** whether a port is free (binds
briefly then immediately closes). This is not a server — it does not hold the socket open.
The validator incorrectly flags it as a violation of `ports.allowed = 0`.

Current exclusion list only contains `portkiller`:

```go
// line 62-63
if strings.Contains(rel, "portkiller") {
    return nil
}
```

### Fix

Add an exclusion for `internal/ports/suggest.go` inside the `WalkDir` callback, right
after the `portkiller` exclusion:

```go
if strings.Contains(rel, "portkiller") {
    return nil
}
// ports/suggest.go probes ports with a bind-then-close; not a server
if strings.HasSuffix(rel, filepath.Join("ports", "suggest.go")) {
    return nil
}
```

### Verification

After fix, `internal/ports/suggest.go` must NOT appear in validator output.
`go run ./scripts/validate-specs.go` must not mention "net.Listen".

---

## Bug 2 — Rule 3: `extractYAMLMethodNames` reads past `events:` section boundary

### File

`scripts/validate-specs.go` — function `extractYAMLMethodNames` (lines 120–135)

### Problem

`specs/ipc.yaml` has two top-level sections: `methods:` and `events:`. The regex
`^\s{2,4}- name:\s*(\w+)` matches `- name:` entries in **both** sections.

Events entries:
```yaml
events:
  - name: server.state.changed   # regex captures "server"
  - name: server.log.batch       # regex captures "server" (duplicate, but "server" is added)
  - name: collision.detected     # regex captures "collision"
```

`(\w+)` stops at the first non-word character (`.`), so "server" and "collision" are
extracted as method names. They don't exist in `app.go`, producing:

```
ipc.yaml methods missing in app.go: [server collision]
```

### Fix

Stop scanning once the `events:` section header is encountered. In `extractYAMLMethodNames`:

```go
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
        // Stop at the events: section — those are not IPC methods
        if strings.TrimSpace(line) == "events:" {
            break
        }
        if m := re.FindStringSubmatch(line); m != nil {
            names = append(names, m[1])
        }
    }
    return names
}
```

### Verification

After fix, "server" and "collision" must NOT appear in the extracted method list.
Running `go run ./scripts/validate-specs.go` must not report Rule 3 failures.

---

## Bug 3 — Rule 4: `extractGoConsts` misses iota-continuation constants

### File

`scripts/validate-specs.go` — function `extractGoConsts` (lines 172–195)

### Problem

`internal/portkiller/state.go` defines all 7 states in a single `const (...)` iota block:

```go
const (
    GracefulSent    State = iota // first
    Polling                      // iota continuation — no "State" on this line
    Released
    ResolveBlocker
    ForceKill
    CrossTargetReport
    UnknownBlocker
)
```

The two regexes in `extractGoConsts` only match lines that explicitly contain the word
`State`:

```go
re  = `^\s+(\w+)\s+State\s*=`   // matches "GracefulSent    State = iota" ✓
re2 = `^\s+(\w+)\s+State\s*$`   // matches nothing — continuation lines have no "State"
```

Result: only `GracefulSent` is extracted. The 6 continuation constants are not found.

The MMD extractor finds 6 states from `port_killer.mmd` (`Polling`, `Released`,
`ResolveBlocker`, `ForceKill`, `CrossTargetReport`, `UnknownBlocker`). None match the
single extracted Go const `["GracefulSent"]`, producing:

```
port_killer.mmd states missing in portkiller/state.go: [Polling Released ResolveBlocker ForceKill CrossTargetReport UnknownBlocker]
```

### Fix

Replace `extractGoConsts` with a block-aware parser that tracks `const (...)` blocks
typed as `State` and collects all identifiers within them:

```go
func extractGoConsts(path string) []string {
    f, err := os.Open(path)
    if err != nil {
        return nil
    }
    defer f.Close()

    // Match the first identifier on a const line (iota leader or continuation).
    // We capture every non-blank, non-comment identifier inside a State const block.
    reIdent  := regexp.MustCompile(`^\s+([A-Za-z]\w*)`)
    reLeader := regexp.MustCompile(`^\s+\w+\s+State\s*=`)

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
        // Enter a State const block when we see "Identifier State = iota"
        if reLeader.MatchString(line) {
            inBlock = true
        }
        if !inBlock {
            continue
        }
        // Skip blank lines and comments inside the block
        if trimmed == "" || strings.HasPrefix(trimmed, "//") {
            continue
        }
        if m := reIdent.FindStringSubmatch(line); m != nil {
            names = append(names, m[1])
        }
    }
    return names
}
```

Remove the old `re2` scanner and the `mustOpen` helper call (they're no longer needed).
`mustOpen` itself can stay if other code uses it, or remove it if `extractGoConsts` was
its only caller.

### Verification

After fix, `extractGoConsts` must return all 7 constants:
`GracefulSent`, `Polling`, `Released`, `ResolveBlocker`, `ForceKill`,
`CrossTargetReport`, `UnknownBlocker`.

Running `go run ./scripts/validate-specs.go` must not report Rule 4 failures.

---

## Final Check

All 3 bugs fixed → run:

```powershell
cd C:\Users\HG\Documents\Phase_yg
go run ./scripts/validate-specs.go
```

Output must be exactly:

```
SSOT validation passed ✓
```

Exit code 0. Nothing else printed to stderr.

Also verify the file still compiles cleanly:

```powershell
go build ./scripts/validate-specs.go
```

(This uses the `//go:build ignore` tag so it won't be included in `go build ./...`,
but running it directly via `go run` is the canonical test.)
