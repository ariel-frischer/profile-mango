# Jcode-like portable profile examples

These profiles model common named-agent workflows without copying or reading any live Jcode configuration:

- `base`: shared route and fail-closed instructions
- `daily`: workspace-writing coding workflow
- `review`: read-only, offline review workflow
- `research`: read-only network research with a portable skill

The binding is an example route identity only. It contains no credentials. Validate a profile with:

```bash
mango validate examples/jcode-like/profiles/daily/profile.yaml \
  --bindings examples/jcode-like/bindings/local.example.yaml --json
```

Render commands must use all three explicit project inputs:

```bash
mango render daily \
  --profiles examples/jcode-like/profiles \
  --resource-root examples/jcode-like \
  --bindings examples/jcode-like/bindings/local.example.yaml \
  --target opencode --target-version 1.18.31 \
  --out ./preview --preview --json
```

All current target renders remain inert and blocked. The examples do not configure authentication, write a target home, start an agent, or prove runtime enforcement.
