# Bus Config SaaS Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drive bus routes from one `config.yaml`, keep legacy home/work (+ optional Bark), add public `GET /watch/bus/q/{hash}` with IP+hash rate limit 4/min.

**Architecture:** Load YAML at startup into hash/legacy maps; register routes from config; in-memory limiter wraps public handlers only; chelaile/QueryRoute unchanged in spirit.

**Tech Stack:** Go 1.26, `gopkg.in/yaml.v3`, stdlib `net/http`, in-process rate limiter.

## Global Constraints

- Public API: no auth; capability = knowing hash
- Public path never pushes Bark
- Legacy paths keep Authorization from config `server.auth_token`
- Rate limit: same IP + same hash ≤ 4 per 60s
- Minimal change: do not rewrite chelaile crypto/protocol
- Spec: `docs/plan/2026-09-20-公交配置化SaaS改造.md`

## File Map

- Create: `config.example.yaml`, `internal/bus/config.go`, `internal/bus/config_test.go`, `internal/bus/limiter.go`, `internal/bus/limiter_test.go`
- Modify: `internal/bus/routes.go`, `internal/bus/auth.go`, `internal/bus/service.go`, `main.go`, `Dockerfile`, `go.mod`

---

### Task 1: Config load + validate

**Files:** Create `internal/bus/config.go`, `internal/bus/config_test.go`, `config.example.yaml`

- [ ] Write failing tests for LoadConfig (valid YAML, duplicate hash, missing fields)
- [ ] Implement LoadConfig + validation
- [ ] Add example `config.yaml` with home/work migrated
- [ ] `go test ./internal/bus/ -run Config`

### Task 2: Rate limiter

**Files:** Create `internal/bus/limiter.go`, `internal/bus/limiter_test.go`

- [ ] Write failing tests (allow 4, block 5th, window reset)
- [ ] Implement Allow(ip, hash) sliding window
- [ ] `go test ./internal/bus/ -run Limiter`

### Task 3: Wire routes + main

**Files:** Modify `routes.go`, `auth.go`, `service.go`, `main.go`, `Dockerfile`

- [ ] Extend RouteConfig; RegisterRoutes from AppConfig
- [ ] Public handler: limiter → query → JSON no Bark
- [ ] Legacy: auth → query → optional Bark
- [ ] main: `-config` flag; Dockerfile COPY config.yaml
- [ ] `go test ./...` and `go build`

---
