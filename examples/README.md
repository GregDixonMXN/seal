# Docket fixtures

- examples/green: `python3 src/write_ok.py` writes src/ok.txt (allowed).
- examples/red: `python3 src/write_env.py` writes .env (denied).

Run each under the compiled policy:

  paldron exec --policy ../../docket-out/paldron.policy.toml -- python3 src/write_ok.py
  paldron exec --policy ../../docket-out/paldron.policy.toml -- python3 src/write_env.py
