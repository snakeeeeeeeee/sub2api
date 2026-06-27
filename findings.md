## Findings
- Account create/update already accepts credentials and extra maps.
- Current OpenAI and Anthropic request builders whitelist inbound headers.
- Gemini and Antigravity builders set headers directly and need explicit hook calls.
- Dev compose file is deploy/docker-compose.dev.yml, not repo root.
- Frontend dependency directory was absent initially; `pnpm --dir frontend install --frozen-lockfile` restored it without changing the lockfile.
- Local `golangci-lint` was absent from PATH. Direct `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0 run ./...` matches the CI lint version and passes.
- A `go install`-built temporary golangci-lint binary can be built with Go 1.25.1 on this machine and then refuses the repo's Go 1.26.4 target; use direct `go run` or a Go 1.26.4-built binary for local Makefile lint.
- Docker compose build currently fails while fetching Docker Hub auth tokens for base image metadata (`postgres:18-alpine`, `golang:1.26.4-alpine`, `node:24-alpine`, `alpine:3.21`), before any project Dockerfile step runs.
