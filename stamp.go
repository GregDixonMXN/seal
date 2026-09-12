package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"
)

// docket stamp freezes one wrapped execution into a signed-style receipt:
// which policies, which exits, what bundle, what verdict. Exit code mirrors
// the verdict: 0 pass, 2 deny, 1 broken. The receipt is always written
// unless its own inputs are unusable.
func runStamp(args []string) (code int) {
	var bundle, command, session, out string
	var gateExit, paldronExit int
	var ballastChangeset, ballastOverlap string
	lock := ""
	for i := 0; i < len(args); i++ {
		str := func() string {
			i++
			if i < len(args) {
				return args[i]
			}
			return ""
		}
		num := func() int {
			n, err := strconv.Atoi(str())
			if err != nil {
				return -1
			}
			return n
		}
		switch args[i] {
		case "--annalist-bundle":
			bundle = str()
		case "--annalist-gate-exit":
			gateExit = num()
		case "--paldron-exit":
			paldronExit = num()
		case "--command":
			command = str()
		case "--session":
			session = str()
		case "--out":
			out = str()
		case "--ballast-changeset":
			ballastChangeset = str()
		case "--ballast-overlap":
			ballastOverlap = str()
		case "--lock":
			lock = str()
		default:
			return fail("unknown flag %q", args[i])
		}
	}
	if command == "" || session == "" || out == "" {
		return fail("--command, --session and --out are required")
	}
	if gateExit < 0 || paldronExit < 0 {
		return fail("--annalist-gate-exit and --paldron-exit must be given as numbers")
	}
	var bundleHash *string
	if bundle != "" {
		h, err := hashBundle(bundle)
		if err != nil {
			return fail("bundle: %v", err)
		}
		bundleHash = &h
	}
	var lockHash *string
	if lock != "" {
		h, err := lockSourceHash(lock)
		if err != nil {
			return fail("lock: %v", err)
		}
		lockHash = &h
	}
	var ballast any
	if ballastChangeset != "" || ballastOverlap != "" {
		m := map[string]any{}
		if ballastChangeset != "" {
			m["changeset"] = ballastChangeset
		}
		if ballastOverlap != "" {
			var v any
			if err := json.Unmarshal([]byte(ballastOverlap), &v); err != nil {
				return fail("ballast-overlap is not JSON: %v", err)
			}
			m["overlap"] = v
		}
		ballast = m
	}
	verdict := "broken"
	switch {
	case gateExit == 0 && paldronExit == 0 && bundleHash != nil:
		verdict = "pass"
	case gateExit == 2 || paldronExit == 2:
		verdict = "deny"
	}
	id := make([]byte, 16)
	if _, err := rand.Read(id); err != nil {
		return fail("random: %v", err)
	}
	receipt := map[string]any{
		"schema":     1,
		"docket_id":  hex.EncodeToString(id),
		"created_at": time.Now().UTC().Format(time.RFC3339),
		"command":    command,
		"annalist": map[string]any{
			"session":       session,
			"gate_exit":     gateExit,
			"bundle_sha256": bundleHash,
		},
		"paldron":            map[string]any{"exit": paldronExit},
		"ballast":            ballast,
		"verdict":            verdict,
		"policy_lock_sha256": lockHash,
	}
	raw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return fail("encode: %v", err)
	}
	if err := os.WriteFile(out, append(raw, '\n'), 0o644); err != nil {
		return fail("write %s: %v", out, err)
	}
	fmt.Printf("stamped %s -> %s\n", receipt["docket_id"], out)
	switch verdict {
	case "pass":
		return 0
	case "deny":
		return 2
	default:
		return 1
	}
}

func fail(format string, args ...any) int {
	fmt.Fprintf(os.Stderr, "docket stamp: "+format+"\n", args...)
	return 1
}

// hashBundle digests a directory deterministically: sorted relative paths,
// each followed by its content. A missing or unreadable dir is an error,
// never an empty hash.
func hashBundle(dir string) (string, error) {
	var files []string
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(files) == 0 {
		return "", fmt.Errorf("empty bundle dir %s", dir)
	}
	sort.Strings(files)
	h := sha256.New()
	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(dir, rel))
		if err != nil {
			return "", err
		}
		h.Write([]byte(rel + "\x00"))
		h.Write(b)
		h.Write([]byte("\x00"))
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// lockSourceHash returns the source sha256 recorded in a docket.lock.json.
func lockSourceHash(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var lock struct {
		Source struct {
			Sha256 string `json:"sha256"`
		} `json:"source"`
	}
	if err := json.Unmarshal(raw, &lock); err != nil {
		return "", err
	}
	if lock.Source.Sha256 == "" {
		return "", fmt.Errorf("no source sha256 in %s", path)
	}
	return lock.Source.Sha256, nil
}
