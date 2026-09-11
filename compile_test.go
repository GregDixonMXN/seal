package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func compileTo(t *testing.T, input string) string {
	t.Helper()
	dir := t.TempDir()
	in := filepath.Join(dir, "docket.toml")
	if err := os.WriteFile(in, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out")
	if err := runCompile([]string{"--in", in, "--out-dir", out}); err != nil {
		t.Fatal(err)
	}
	return out
}

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const exampleToml = `schema = 1

[paths]
allow = ["src/", "docs/", "tests/"]
deny = [".env", ".env.*", "*.pem", "**/secrets/**"]
max_files_changed = 80

[exec]
allow_network = false
allow_binaries = ["python3", "git", "ls", "cat"]
require_os_isolation = true
timeout_sec = 30
fail_on_secret = true
`

func TestCompileExample(t *testing.T) {
	out := compileTo(t, exampleToml)
	ap := read(t, filepath.Join(out, "annalist.policy.toml"))
	for _, want := range []string{`allow_paths = ["src/", "docs/", "tests/"]`, "max_files_changed = 80", "fail_on_secret = true"} {
		if !strings.Contains(ap, want) {
			t.Fatalf("annalist policy missing %q:\n%s", want, ap)
		}
	}
	pp := read(t, filepath.Join(out, "paldron.policy.toml"))
	for _, want := range []string{"allow_network = false", "require_os_isolation = true", "timeout_sec = 30"} {
		if !strings.Contains(pp, want) {
			t.Fatalf("paldron policy missing %q:\n%s", want, pp)
		}
	}
	lock := read(t, filepath.Join(out, "docket.lock.json"))
	if !strings.Contains(lock, `"schema": 1`) || !strings.Contains(lock, "annalist.policy.toml") {
		t.Fatalf("lock malformed:\n%s", lock)
	}
}

func TestUnknownKeyIsError(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "docket.toml")
	os.WriteFile(in, []byte("schema = 1\nsneaky = true\n"), 0o644)
	if err := runCompile([]string{"--in", in, "--out-dir", filepath.Join(dir, "out")}); err == nil {
		t.Fatal("unknown key silently accepted")
	}
}

func TestBadSectionIsError(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "docket.toml")
	os.WriteFile(in, []byte("schema = 1\n[billing]\nprice = 5\n"), 0o644)
	if err := runCompile([]string{"--in", in, "--out-dir", filepath.Join(dir, "out")}); err == nil {
		t.Fatal("unknown section silently accepted")
	}
}

func TestMissingSchemaIsError(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "docket.toml")
	os.WriteFile(in, []byte("[paths]\nallow = [\"src/\"]\n"), 0o644)
	if err := runCompile([]string{"--in", in, "--out-dir", filepath.Join(dir, "out")}); err == nil {
		t.Fatal("missing schema silently accepted")
	}
}
