# Docket — optional glue for the suite

One policy file (`docket.toml`) that compiles into the policy files
Annalist and Paldron enforce. No dashboard, no runtime, no billing.

- Annalist records what your agent did (flight recorder).
- Paldron decides whether it may run (policy gate + sandbox).
- Docket keeps their policies from drifting apart.

Each tool works alone. Docket only helps them work together.

## Use

docket compile --in docket.toml --out-dir ./docket-out
annalist run -- paldron exec --policy docket-out/paldron.policy.toml -- <command>

See examples/green (passes) and examples/red (.env denied).
`docket stamp` + the GitHub Action land after a stranger's PR goes red.
