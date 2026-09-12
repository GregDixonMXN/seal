package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func bundleDir(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func stamp(t *testing.T, args ...string) int {
	t.Helper()
	if code := runStamp(args); code < 0 {
		t.Fatalf("runStamp returned %d", code)
	} else {
		return code
	}
	return -1
}

func receipt(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var r map[string]any
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestStampPassRoundTrip(t *testing.T) {
	dir := t.TempDir()
	bundle := bundleDir(t, map[string]string{"session.json": `{"id":1}`})
	out := filepath.Join(dir, "receipt.json")
	code := stamp(t, "--command", "paldron exec -- true",
		"--session", "163", "--out", out,
		"--annalist-gate-exit", "0", "--paldron-exit", "0",
		"--annalist-bundle", bundle)
	if code != 0 {
		t.Fatalf("pass stamp exit = %d, want 0", code)
	}
	if got := receipt(t, out)["verdict"]; got != "pass" {
		t.Fatalf("verdict = %v, want pass", got)
	}
	if code := runVerify([]string{"--bundle", bundle, out}); code != 0 {
		t.Fatalf("verify --bundle exit = %d, want 0", code)
	}
}

func TestStampDeny(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "receipt.json")
	code := stamp(t, "--command", "paldron exec -- python3 src/write_env.py",
		"--session", "164", "--out", out,
		"--annalist-gate-exit", "0", "--paldron-exit", "2")
	if code != 2 {
		t.Fatalf("deny stamp exit = %d, want 2", code)
	}
	if got := receipt(t, out)["verdict"]; got != "deny" {
		t.Fatalf("verdict = %v, want deny", got)
	}
	if code := runVerify([]string{out}); code != 0 {
		t.Fatalf("verify exit = %d, want 0", code)
	}
}

func TestStampBroken(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "receipt.json")
	code := stamp(t, "--command", "paldron exec -- boom",
		"--session", "165", "--out", out,
		"--annalist-gate-exit", "1", "--paldron-exit", "1")
	if code != 1 {
		t.Fatalf("broken stamp exit = %d, want 1", code)
	}
	if got := receipt(t, out)["verdict"]; got != "broken" {
		t.Fatalf("verdict = %v, want broken", got)
	}
}

func TestVerifyBundleMismatch(t *testing.T) {
	dir := t.TempDir()
	bundle := bundleDir(t, map[string]string{"session.json": `{"id":1}`})
	out := filepath.Join(dir, "receipt.json")
	stamp(t, "--command", "x", "--session", "1", "--out", out,
		"--annalist-gate-exit", "0", "--paldron-exit", "0",
		"--annalist-bundle", bundle)
	if err := os.WriteFile(filepath.Join(bundle, "session.json"), []byte(`{"id":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runVerify([]string{"--bundle", bundle, out}); code == 0 {
		t.Fatal("verify passed on tampered bundle, want failure")
	}
}

func TestVerifyWrongVerdict(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "receipt.json")
	stamp(t, "--command", "x", "--session", "1", "--out", out,
		"--annalist-gate-exit", "0", "--paldron-exit", "0",
		"--annalist-bundle", bundleDir(t, map[string]string{"s": "x"}))
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	forged := strings.Replace(string(raw), `"pass"`, `"deny"`, 1)
	if err := os.WriteFile(out, []byte(forged), 0o644); err != nil {
		t.Fatal(err)
	}
	if code := runVerify([]string{out}); code == 0 {
		t.Fatal("verify passed on forged verdict, want failure")
	}
}

func TestStampRequiresInputs(t *testing.T) {
	if code := runStamp([]string{"--command", "x"}); code == 0 {
		t.Fatal("stamp with missing flags passed, want failure")
	}
	if code := runVerify([]string{filepath.Join(t.TempDir(), "missing.json")}); code == 0 {
		t.Fatal("verify of missing receipt passed, want failure")
	}
}

func TestHelp(t *testing.T) {
	if code := runStamp([]string{"--help"}); code != 0 {
		t.Fatalf("stamp --help exit = %d, want 0", code)
	}
	if code := runVerify([]string{"--help"}); code != 0 {
		t.Fatalf("verify --help exit = %d, want 0", code)
	}
}
