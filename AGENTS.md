# AGENTS.md

## Global Rules

- Do not open PRs unless requested.
- Make minimal, scoped changes.
- Go version `1.25` exists. Do not try to downgrade.
- Do not introduce new dependencies unless explicitly asked.
- Do not use emojis unless explicitly asked.

## Repo Structure

- The repo follows https://github.com/golang-standards/project-layout
- Unit tests live next to code.
- e2e tests live in `test/`

## Tooling

- Use `task` for workflows when possible.
- Use `uv` for Python if needed.
- Avoid `curl | bash` unless absolutely necessary.
- Prefer official Github Actions, then actions from reputable organizations.
- Do not use Github Actions from random Github users.

## Testing

- Code must compile.
- Existing tests must pass.

## Communication

- Summarize changes and files touched.
- Ask if unsure.
