# AGENTS.md

## Global Rules

- Do not open PRs unless requested.
- Make minimal, scoped changes.
- Go version `1.25` exists. Do not try to downgrade.
- The `kubesnake` binary should remain a single static, portable binary.
- Do not introduce new dependencies unless explicitly asked.
- Do not use emojis unless explicitly asked.

## Repo Structure

- The repo follows https://github.com/golang-standards/project-layout
- Unit tests live next to code.
- e2e tests live in `test/`

## Tooling

- Use `task` for workflows when possible.
- Obtain the `task` binary for `Taskfile.yml` if you don’t have it already.
- Use `uv` for Python if needed.
- Avoid `curl | bash` unless absolutely necessary.
- Prefer official Github Actions, then actions from reputable organizations.
- Do not use Github Actions from random Github users.

## Testing

- Code must compile.
- Existing tests must pass.
- Obtain Golang `1.25` if you don’t have it already.
- Run `task lint`, `task build`, and `task test:unit` locally before pushing to GitHub, to keep iteration cycles faster.
- The `task test:e2e` command only works if Docker-in-Docker works, Docker is installed, and k3d is installed; this probably isn’t the case for hosted agent sandboxes.
- Github Actions will run the e2e test suite automatically for you to get feedback and iterate on your implementation.

## Communication

- Summarize changes and files touched.
- Ask if unsure.
