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
`seal stamp` + the GitHub Action land after a stranger's PR goes red.
