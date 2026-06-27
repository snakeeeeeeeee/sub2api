## Goal
Implement account-level upstream header templates with minimal fork surface.

## Phases
1. Backend template resolver and tests - complete
2. Wire resolver into upstream request builders - complete
3. Frontend editor component and modal integration - complete
4. Compose image tag update - complete
5. Verification and cleanup - complete

## Constraints
- No database migration; use account.extra.upstream_headers.
- Keep changes localized for easier upstream merges.
- Do not allow custom headers to override auth/protocol headers.

## Verification Notes
- Backend targeted tests for helper and builder paths pass.
- `go test ./...` passes.
- `go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.9.0 run ./...` passes with 0 issues.
- `pnpm --dir frontend run typecheck` passes after installing frontend dependencies from lockfile.
- Account editor Vitest coverage for upstream header editor and edit modal integration passes.
- `docker compose config` confirms `SUB2API_DEV_IMAGE=sub2api:header-template-dev` renders into `services.sub2api.image`.
- Docker compose build is blocked before project build by Docker Hub auth token timeout while loading base image metadata.
