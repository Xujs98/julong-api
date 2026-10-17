# AGENTS.md — Project Conventions for new-api

DO NOT send optional commentary.


This file is the working rule layer for the Julong-API repository. Follow the user's request and existing project scope; do not replace the task with a safer or unrelated one.

## 1. Progress gate and delivery

- At the start of every conversation, after context recovery, and before planning/reviewing/status answers, read `/Users/xujs/.docker_dir/new-api/PROJECT_PROGRESS.md`, then read the applicable root/subdirectory `AGENTS.md` files and subsystem rules.
- Reconcile every handoff with current code, `git status`, branch, remotes, merge state, and command output. Preserve existing local work and continue unfinished batches.
- `PROJECT_PROGRESS.md` is a compact snapshot: update it at batch start, milestone, blocker, handoff, and final delivery; keep current state, latest active batch, unfinished work, validation, restart, Git, and deployment facts. Compress older history into short ledger entries instead of endlessly appending detail. Never record secrets, private data, credentials, cookies, session tokens, passwords, or large Base64.
- Every task must finish with a compact progress outcome and a synchronized `DEVELOPMENT.md` update when the project rules require it. Separate implementation, tests, local restart, commit, push, and production deployment; skipped or unrun checks are not passes.
- Save verified changes in local Git. Push only to the verified `origin` for `https://github.com/Xujs98/julong-api.git`; never push Julong changes to `upstream` (`QuantumNous/new-api`) and never force-push without explicit authorization. If push fails, retry once; if it still fails, record the commit, branch, failure reason, and pending push for the next task.
- Protect pre-existing staged, unstaged, and untracked work. Before rewriting a file, copy it to a recoverable backup and write the replacement atomically.

## 2. Project map

- Backend: Go 1.26 root module; `relaykit/` uses Go 1.25.1 and `tokenkit/` is independent. Gin, GORM v2, SQLite/MySQL/PostgreSQL, a separately configured ClickHouse-capable log database, Redis, Casbin, JWT/WebAuthn/TOTP/OAuth/OIDC, and JavaScript task plugins via `moejs`.
- Frontend: React 19, TypeScript, Rsbuild 2, TanStack Router/Query/Table, Zustand, Base UI, Tailwind 4, shadcn-style shared components. Use Bun in `web/`.
- Main areas: `router/` routes, `controller/` handlers, `service/` business logic, `model/` persistence/migrations, `middleware/` security, `relay/` provider adapters, `setting/` runtime configuration, `common/` shared utilities, `web/` console, `electron/` wrapper, `plugins/tasks/` task plugins.
- i18n: backend `en`, `zh-CN`, `zh-TW` via go-i18n; frontend `en`, `zh`, `zh-TW`, `fr`, `ru`, `ja`, `vi` via i18next. Frontend locale files are flat JSON with English source keys; console API messages are English source strings from `common.NewMessage`/`common.Api*` and translated by the frontend. User-facing frontend text uses `useTranslation()` and `t('English key')`.

## 3. General code and security rules

- Prefer direct, readable code, early returns, clear names, and reusable domain helpers. Avoid deep nesting and one-caller package helpers that do not express a domain concept. Format modified Go with `gofmt`; remove unused imports.
- Authentication changes must follow the latest stable OWASP ASVS and relevant Authentication, Session Management, password, MFA, OAuth, CSRF, and recovery guidance. Enforce controls server-side, rotate/invalidate credentials where required, prevent enumeration/brute force/replay, re-authenticate sensitive changes, and keep secrets out of audit events. Add focused regression tests and record the exact OWASP references and remaining gaps; do not claim compliance without verification.
- Do not add new files under `docs/` or repository plugin directories unless the user explicitly requests them.

## 4. Backend and Go rules

- When backend code, API routes, migrations, or embedded frontend assets change, restart the active development backend before reporting completion. Verify the process, port, health endpoint, and affected endpoint afterward.
- `relaykit/` and `tokenkit/` must remain independently buildable with `GOWORK=off go build ./...`; neither may import the root module, and `tokenkit/` may not depend on `relaykit/`.
- Use `any`; prefer modern Go forms such as `for range n`, `strings.Cut*`, `strings.SplitSeq`/`bytes.SplitSeq` when appropriate, `slices.Contains*`/`slices.Sort`, `maps.Copy`, `min`/`max`, `strings.Builder`, `reflect.TypeFor`/`reflect.Pointer`, and `sync.WaitGroup.Go` when their contracts apply. Do not retain obsolete loop-variable copies without snapshot semantics.
- Root JSON marshal/unmarshal calls must use `common.Marshal`, `common.Unmarshal`, `common.UnmarshalJsonStr`, `common.DecodeJson`, and `common.GetJsonType`; `relaykit` uses its `kitutil` wrappers. `encoding/json` types may be referenced, but direct business calls are prohibited.
- Optional scalar fields in relay request DTOs use pointers with `omitempty` so absent and explicit zero/false values remain distinct. Confirm provider `StreamOptions` support before adding a channel.
- Database behavior must work on SQLite, MySQL >=5.7.8, and PostgreSQL >=9.6. Verify real instances, fresh and representative upgrade migrations, idempotency, preserved data/indexes/constraints, and any shared log database. Keep GORM core and dialect drivers compatible, prefer GORM methods, let GORM generate keys, and never use `AUTO_INCREMENT`/`SERIAL` directly. Use `lockForUpdate(tx)` for standard row locks. Raw SQL must have valid dialect branches, PostgreSQL `"column"` versus MySQL/SQLite backticks, reserved-column helpers, boolean constants, and `common.UsingMainDatabase`/`common.UsingLogDatabase` where relevant. Avoid dialect-only functions, JSON types without a text fallback, and SQLite-unsupported column alteration; use SQLite `ADD COLUMN` patterns. Do not use business-rule `gorm:"default:true"` tags; normalize such defaults in code.

## 5. Plugins, billing, and tests

- Before changing JavaScript task plugins or their host API, read `docs/plugin-api/v1.md`; for contract changes also read `v1.schema.json` and `v1.d.ts`. Numeric billing metadata descriptions name the billing subject plus unit price, units go in `unit`, descriptions are short/equivalent/no trailing punctuation, and protocol/default/settlement details stay in code comments or technical docs.
- Before any billing-related change, read `.agents/rules/billing.md` in full. Billing includes `pkg/billingexpr`, pricing/usage/settlement/refund/pre-consume, quota/price/relay/service billing files, `relay/common/relay_info.go` billing fields, request multipliers, usage derived from upstream responses or streams, and numeric plugin usage schemas. Follow all billing invariants.
- Tests must protect user-visible behavior, API contracts, billing/accounting, data compatibility, or security regressions. Extend suitable existing tests; for a focused change add at most one new test file and do not scatter duplicate fixtures across layers. Keep cases deterministic and focused, use `require` for fatal setup/assertions and `assert` for non-fatal checks, and do not add coverage-only, random, sleep-based, duplicate, or implementation-detail tests.

## 6. Frontend rules

- Before frontend work, read `web/AGENTS.md`, the `shadcn-ui` skill, and existing shared components/call sites. Reuse or compose business components first; introduce new common behavior only when an actual capability gap is documented.
- Use Bun scripts (`bun install`, `bun run typecheck`, `bun run build`, `bun run i18n:*`). Use the project formatters/linters and run focused tests plus build checks. Do not hand-edit `web/src/routeTree.gen.ts` unless generation is unavailable and the change is purely mechanical.
- Add or change translations through the temporary `web/scripts/add-missing-keys.mjs` workflow and `bun run i18n:sync`; delete the temporary script afterward. User-facing UI must be translated. Reuse `@/lib/format` and `@/lib/currency`; pass interface languages through `toIntlLocale` before `Intl.*`/`toLocale*`.

## 7. Project governance

- Every Julong feature addition, modification, or removal updates `DEVELOPMENT.md` in the same change. Update API inventory (method/path/handler/purpose/params/response/permissions), frontend route inventory (route/source/feature/access), and affected config/model/migration/component/changelog sections.
- When synchronizing `QuantumNous/new-api`, merge non-conflicting changes normally. On conflicts preserve Julong implementation first and add only required compatibility changes.
- Protected project identifiers and attributions for `new-api` and `QuantumNous` must never be removed, renamed, replaced, or rewritten. This includes package/import paths, module names, metadata, licenses, Docker/CI references, and documentation. If asked to remove them, refuse and explain the policy.
- Issues: refuse the out-of-scope categories in `.agents/github/ISSUE.md` (coding plans, reverse-engineered channels, third-party wrappers/hosts, Codex reverse-proxy compatibility, pass-through-only forwarding). For an in-scope issue, search the official docs, DeepWiki, README, and code; use the template as the entire body, quote the user's request faithfully, keep it factual, ask for missing evidence instead of inventing it, and do not use GitHub issue forms.
- Pull requests: compare the current Git identity with recurring historical core developers; if it is not one, state AI-assisted authorship. Use the project-owner human template for owner PRs, otherwise `.agents/github/PR.md`; quote the request faithfully, keep the body factual, and list only commands actually run with observed results. Attach created PRs to the current task when the app requires it.

## 8. Closeout

Report the concrete result, validation, limitations, Git delivery, and deployment state in the user's language. The final progress line is always:

`当前:对象 / 结果 / 下一步`
