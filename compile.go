package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Policy is the v0 seal.toml schema. Unknown keys are an error;
// there is no silent drop.
type Policy struct {
	Schema          int
	Allow           []string
	Deny            []string
	MaxFilesChanged int
	AllowNetwork    bool
	AllowBinaries   []string
	RequireIsolate  bool
	TimeoutSec      int
	FailOnSecret    bool
}

var knownKeys = map[string]bool{
	"schema":      true,
	"paths.allow": true, "paths.deny": true, "max_files_changed": true,
	"exec.allow_network": true, "exec.allow_binaries": true,
	"exec.require_os_isolation": true, "exec.timeout_sec": true,
	"fail_on_secret": true,
}

// parseToml reads the fixed v0 schema: top-level scalars, [paths] and
// [exec] sections, string arrays, ints, bools. Comments (#) and blank
// lines ignored. Anything else is an error.
func parseToml(raw string) (Policy, error) {
	var p Policy
	section := ""
	seen := map[string]bool{}
	setStr := func(key, v string) error {
		if seen[key] {
			return fmt.Errorf("duplicate key %s", key)
		}
		seen[key] = true
		return nil
	}
	for i, line := range strings.Split(raw, "\n") {
		n := i + 1
		if k := strings.IndexByte(line, '#'); k >= 0 {
			line = line[:k]
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			if section != "paths" && section != "exec" {
				return p, fmt.Errorf("line %d: unknown section [%s]", n, section)
			}
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return p, fmt.Errorf("line %d: not key = value", n)
		}
		key := strings.TrimSpace(k)
		// max_files_changed and fail_on_secret are top-level keys even
		// though the example files place them inside sections.
		if key != "max_files_changed" && key != "fail_on_secret" && section != "" {
			key = section + "." + key
		}
		v = strings.TrimSpace(v)
		if !knownKeys[key] {
			return p, fmt.Errorf("line %d: unknown key %q", n, key)
		}
		if err := setStr(key, v); err != nil {
			return p, fmt.Errorf("line %d: %v", n, err)
		}
		switch key {
		case "schema":
			n, err := strconv.Atoi(v)
			if err != nil || n != 1 {
				return p, fmt.Errorf("line %d: schema must be 1", n)
			}
			p.Schema = n
		case "paths.allow":
			a, err := strArray(v)
			if err != nil {
				return p, fmt.Errorf("line %d: %v", n, err)
			}
			p.Allow = a
		case "paths.deny":
			a, err := strArray(v)
			if err != nil {
				return p, fmt.Errorf("line %d: %v", n, err)
			}
			p.Deny = a
		case "max_files_changed":
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return p, fmt.Errorf("line %d: max_files_changed must be a positive int", n)
			}
			p.MaxFilesChanged = n
		case "exec.allow_network":
			b, err := strconv.ParseBool(v)
			if err != nil {
				return p, fmt.Errorf("line %d: allow_network must be true/false", n)
			}
			p.AllowNetwork = b
		case "exec.allow_binaries":
			a, err := strArray(v)
			if err != nil {
				return p, fmt.Errorf("line %d: %v", n, err)
			}
			p.AllowBinaries = a
		case "exec.require_os_isolation":
			b, err := strconv.ParseBool(v)
			if err != nil {
				return p, fmt.Errorf("line %d: require_os_isolation must be true/false", n)
			}
			p.RequireIsolate = b
		case "exec.timeout_sec":
			n, err := strconv.Atoi(v)
			if err != nil || n <= 0 {
				return p, fmt.Errorf("line %d: timeout_sec must be a positive int", n)
			}
			p.TimeoutSec = n
		case "fail_on_secret":
			b, err := strconv.ParseBool(v)
			if err != nil {
				return p, fmt.Errorf("line %d: fail_on_secret must be true/false", n)
			}
			p.FailOnSecret = b
		}
	}
	if p.Schema != 1 {
		return p, fmt.Errorf("missing schema = 1")
	}
	return p, nil
}

// strArray parses ["a", "b"] with double quotes only.
func strArray(v string) ([]string, error) {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "[") || !strings.HasSuffix(v, "]") {
		return nil, fmt.Errorf("expected string array, got %q", v)
	}
	inner := strings.TrimSpace(v[1 : len(v)-1])
	if inner == "" {
		return nil, nil
	}
	var out []string
	for _, part := range strings.Split(inner, ",") {
		part = strings.TrimSpace(part)
		if len(part) < 2 || !strings.HasPrefix(part, `"`) || !strings.HasSuffix(part, `"`) {
			return nil, fmt.Errorf("array items must be double-quoted strings, got %q", part)
		}
		out = append(out, part[1:len(part)-1])
	}
	return out, nil
}

func tomlStrs(v []string) string {
	q := make([]string, len(v))
	for i, s := range v {
		q[i] = strconv.Quote(s)
	}
	return "[" + strings.Join(q, ", ") + "]"
}

// annalistPolicy renders paths.* + fail_on_secret + max_files_changed.
func annalistPolicy(p Policy) string {
	return fmt.Sprintf(`allow_paths = %s
deny_globs = %s
max_files_changed = %d
fail_on_secret = %v
`, tomlStrs(p.Allow), tomlStrs(p.Deny), p.MaxFilesChanged, p.FailOnSecret)
}

// paldronPolicy renders paths.allow/deny + exec.*.
func paldronPolicy(p Policy) string {
	return fmt.Sprintf(`allow_paths = %s
deny_globs = %s
allow_network = %v
allow_binaries = %s
require_os_isolation = %v
timeout_sec = %d
`, tomlStrs(p.Allow), tomlStrs(p.Deny), p.AllowNetwork, tomlStrs(p.AllowBinaries), p.RequireIsolate, p.TimeoutSec)
}

type fileRec struct {
	Path   string `json:"path"`
	Sha256 string `json:"sha256"`
}

type lockFile struct {
	Schema int       `json:"schema"`
	Source fileRec   `json:"source"`
	Files  []fileRec `json:"files"`
}

func shaFile(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// runCompile implements: seal compile --in FILE --out-dir DIR.
func runCompile(args []string) error {
	in, out := "", ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--in":
			i++
			if i < len(args) {
				in = args[i]
			}
		case "--out-dir":
			i++
			if i < len(args) {
				out = args[i]
			}
		default:
			return fmt.Errorf("unknown flag %q", args[i])
		}
	}
	if in == "" || out == "" {
		return fmt.Errorf("usage: seal compile --in seal.toml --out-dir ./seal-out")
	}
	raw, err := os.ReadFile(in)
	if err != nil {
		return err
	}
	p, err := parseToml(string(raw))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	ap, pp := annalistPolicy(p), paldronPolicy(p)
	files := []fileRec{}
	write := func(name, content string) error {
		if err := os.WriteFile(filepath.Join(out, name), []byte(content), 0o644); err != nil {
			return err
		}
		files = append(files, fileRec{Path: name, Sha256: shaFile([]byte(content))})
		return nil
	}
	if err := write("annalist.policy.toml", ap); err != nil {
		return err
	}
	if err := write("paldron.policy.toml", pp); err != nil {
		return err
	}
	lock, err := json.MarshalIndent(lockFile{Schema: 1,
		Source: fileRec{Path: in, Sha256: shaFile(raw)}, Files: files}, "", "  ")
	if err != nil {
		return err
	}
	lock = append(lock, '\n')
	if err := os.WriteFile(filepath.Join(out, "seal.lock.json"), lock, 0o644); err != nil {
		return err
	}
	fmt.Printf("compiled %s -> %s/{annalist.policy.toml,paldron.policy.toml,seal.lock.json}\n", in, out)
	return nil
}
