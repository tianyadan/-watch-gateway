# Design: Bus Config SaaS (watch-gateway)

Date: 2026-09-20  
Status: Approved in discussion; awaiting user review of this written spec

## Summary

Evolve personal hard-coded bus gateway into a small multi-caller SaaS driven by one YAML file. Keep legacy Watch endpoints with auth + optional Bark. Add public hash endpoints that return JSON only, rate-limited to 4 requests/minute per IP+hash.

## Decisions

1. **Auth for public API:** none; knowledge of hash is the capability (Option A).
2. **Bark:** keep on legacy paths only when `bark: true`; public `q/{hash}` never pushes.
3. **Config:** single `config.yaml`; migrate home/work into it with `legacy` + `bark` flags.
4. **Rate limit:** in-memory, key=`IP|hash`, window=60s, limit=4; no Redis this phase.
5. **Minimal change:** keep chelaile client, crypto, QueryRoute core; replace hard-coded routes with config loader + route registration.

## Config sketch

See `docs/plan/2026-09-20-公交配置化SaaS改造.md` §4.1 for full YAML.

## API

| Path | Auth | Bark | Limit |
|------|------|------|-------|
| `GET /watch/bus/home\|work` (from `legacy`) | config `auth_token` | if `bark` | none |
| `GET /watch/bus/q/{hash}` | none | never | 4/min per IP+hash |
| `GET /health` | none | — | none |

## Out of scope

User accounts, billing, admin UI, hot reload, auto hash generation, distributed rate limit, chelaile protocol changes.

## Implementation entry

Full Chinese design + TODO: `docs/plan/2026-09-20-公交配置化SaaS改造.md`
