// Command seal compiles one org policy (seal.toml) into the policy
// files Annalist and Paldron actually enforce. Optional glue: every
// tool works alone; Seal only keeps their policies from drifting.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: seal <compile|stamp|verify> [flags]")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "compile":
		if err := runCompile(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "seal:", err)
			os.Exit(1)
		}
	case "stamp":
		os.Exit(runStamp(os.Args[2:]))
	case "verify":
		os.Exit(runVerify(os.Args[2:]))
	default:
		fmt.Fprintln(os.Stderr, "seal: unknown command", os.Args[1])
		os.Exit(1)
	}
}
