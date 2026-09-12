package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// docket verify checks a receipt: schema, required fields, verdict
// consistency, and — with --bundle — that the bundle still hashes to
// the recorded digest. Exit 0 valid, 1 broken.
func runVerify(args []string) int {
	var file, bundle string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--bundle":
			i++
			if i < len(args) {
				bundle = args[i]
			}
		default:
			if file == "" && args[i][0] != '-' {
				file = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "docket verify: unknown flag %q\n", args[i])
				return 1
			}
		}
	}
	if file == "" {
		fmt.Fprintln(os.Stderr, "docket verify: receipt path required")
		return 1
	}
	raw, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "docket verify: %v\n", err)
		return 1
	}
	var r struct {
		Schema    int    `json:"schema"`
		DocketID  string `json:"docket_id"`
		CreatedAt string `json:"created_at"`
		Command   string `json:"command"`
		Annalist  *struct {
			Session    string  `json:"session"`
			GateExit   int     `json:"gate_exit"`
			BundleHash *string `json:"bundle_sha256"`
		} `json:"annalist"`
		Paldron *struct {
			Exit int `json:"exit"`
		} `json:"paldron"`
		Verdict string `json:"verdict"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		fmt.Fprintf(os.Stderr, "docket verify: bad JSON: %v\n", err)
		return 1
	}
	bad := func(format string, args ...any) int {
		fmt.Fprintf(os.Stderr, "docket verify: "+format+"\n", args...)
		return 1
	}
	if r.Schema != 1 {
		return bad("schema %d, want 1", r.Schema)
	}
	if r.DocketID == "" || r.CreatedAt == "" || r.Command == "" {
		return bad("docket_id, created_at and command are required")
	}
	if r.Annalist == nil || r.Annalist.Session == "" || r.Paldron == nil {
		return bad("annalist{session,gate_exit} and paldron{exit} are required")
	}
	want := "broken"
	switch {
	case r.Annalist.GateExit == 0 && r.Paldron.Exit == 0 && r.Annalist.BundleHash != nil:
		want = "pass"
	case r.Annalist.GateExit == 2 || r.Paldron.Exit == 2:
		want = "deny"
	}
	if r.Verdict != want {
		return bad("verdict %q inconsistent with exits (want %q)", r.Verdict, want)
	}
	if bundle != "" {
		if r.Annalist.BundleHash == nil {
			return bad("no bundle digest recorded")
		}
		h, err := hashBundle(bundle)
		if err != nil {
			return bad("bundle: %v", err)
		}
		if h != *r.Annalist.BundleHash {
			return bad("bundle digest mismatch: have %s want %s", h, *r.Annalist.BundleHash)
		}
	}
	fmt.Printf("verify ok: %s verdict=%s\n", r.DocketID, r.Verdict)
	return 0
}
