# Seal — optional glue for the suite

One policy file (`seal.toml`) that compiles into the policy files
Annalist and Paldron enforce. No dashboard, no runtime, no billing.

- Annalist records what your agent did (flight recorder).
- Paldron decides whether it may run (policy gate + sandbox).
- Seal keeps their policies from drifting apart.

Each tool works alone. Seal only helps them work together.

## Use

seal compile --in seal.toml --out-dir ./seal-out
annalist run -- paldron exec --policy seal-out/paldron.policy.toml -- <command>

See examples/green (passes) and examples/red (.env denied).

## Stamp and verify

`seal stamp` freezes one wrapped execution into a receipt (exits,
bundle digest, verdict) and mirrors the verdict in its own exit code:
0 pass, 2 deny, 1 broken. `seal verify` rechecks it, optionally
re-hashing the bundle:

seal stamp --command "paldron exec -- python3 src/write_ok.py" --session 163 --out receipt.json \
  --annalist-gate-exit 0 --paldron-exit 0 --annalist-bundle ./annalist-bundle
seal verify --bundle ./annalist-bundle receipt.json

## Action

`.github/actions/seal-run` compiles `seal.toml` with a pinned seal
and fails the job when the committed `seal-out/` drifted. The
`suite demo` workflow proves it: green compiles, stamps, and verifies;
red tampers a bundle and `seal verify` refuses it.
