# Antigravity Knowledge Item: Gen Con Planner Agent Workflow Standards

## Overview
This Knowledge Item enforces mandatory operational standards, testing requirements, documentation synchronization, and multi-persona review protocols for all Antigravity agent sessions working on the Gen Con Planner repository.

## 1. Mandatory Pre-Flight Verification Checklist
Before completing any request, requesting user review, or concluding a session, the Antigravity agent MUST ensure the following four requirements are met:

### A. Run All Automated Tests
Execute the full test suite across backend and frontend layers:
- **Backend Unit & Integration Tests**:
  ```bash
  go test -v ./internal/background/... ./internal/bgg/... ./internal/events/...
  go test -tags=integration -v ./internal/postgres/... ./internal/api/...
  ```
- **Frontend Unit & Component Tests**:
  ```bash
  cd ui && npm test -- --watch=false
  ```
- **Frontend End-to-End Tests**:
  ```bash
  cd ui && npx playwright test
  ```

### B. Introduce New Automated Tests
Every functional change or bug fix MUST include new automated test coverage:
- **Backend**: Add unit tests (`*_test.go`) for business logic and Testcontainers integration tests for database queries/triggers.
- **Frontend**: Add Vitest component/service specs (`*.spec.ts`) and Playwright E2E user journey tests in `ui/e2e/`.

### C. Keep User Documentation Up-to-Date
Verify and update user documentation in `docs/` to reflect any changes to interaction flows, UI navigation, party mode mechanics, or wishlist prioritization:
- `docs/01-welcome-to-gencon-planner.md`
- `docs/02-browsing-and-searching-events.md`
- `docs/03-account-setup-and-preferences.md`

### D. Run Appropriate Linting Tools
Maintain code quality and formatting consistency:
- **Go Backend**: `go vet ./...` and `gofmt -s -w .`
- **Angular Frontend**: `cd ui && npx prettier --check .` and `cd ui && npm run ng -- lint`

---

## 2. Multi-Persona Review Protocol

Before finalizing any task or requesting user approval, the Antigravity agent MUST enter a dedicated **Verification & Review Mode**. The agent must sequentially simulate the five expert engineering personas below in the exact order specified, auditing its own proposed changes against each persona's checklist and proactively applying any necessary fixes.

**Crucial Workflow Note**: The **Experienced Security Engineer** review is intentionally placed as the **final step** in the protocol. This ensures that any code adjustments, logging additions, UI tweaks, or build optimizations introduced during the QA, UX, DevOps, or Reliability reviews are rigorously audited for vulnerabilities before final sign-off.

### Persona 1: Experienced QA Engineer
**Focus**: Test Coverage, Edge Cases, and System Testability.
- **Checklist**:
  - [ ] Are all new or modified business logic paths covered by automated unit/component tests?
  - [ ] Have edge cases (e.g., empty states, boundary conditions, malformed input, network timeouts) been explicitly tested?
  - [ ] Is the test suite deterministic? (e.g., no flaky `time.Sleep` calls; proper use of injected clocks and Testcontainers seed fixtures).
  - [ ] Did all existing backend, frontend, and E2E test suites execute and pass successfully?

### Persona 2: Experienced UX Engineer
**Focus**: Visual Consistency, Responsive Design, and Premium Interaction Standards.
- **Checklist**:
  - [ ] **Visual Consistency**: Does the change strictly adhere to the established design system, typography (Google Fonts), color palettes, and glassmorphism/modern aesthetics?
  - [ ] **Layout & Responsiveness**: Are flexbox/grid layouts correctly structured? Ensure split-pane containers, navigation bars, and footers maintain perfect alignment without unwanted whitespace or overflow issues across mobile and desktop viewports.
  - [ ] **Micro-Interactions & Feedback**: Are interactive elements (buttons, star toggles, forms) equipped with clear hover/active states, smooth transitions, and appropriate loading/disabled indicators?
  - [ ] **Behavioral Continuity**: Does the new feature maintain familiar interaction patterns and navigation flows established across the rest of the Gen Con Planner application?

### Persona 3: Experienced DevOps Engineer
**Focus**: Build Optimization, Test Execution Efficiency, and CI/CD Performance.
- **Checklist**:
  - [ ] **Build Caching & Layering**: Are Dockerfile layers optimized for maximum caching? (e.g., copying `go.mod`/`go.sum` or `package.json`/`package-lock.json` before full source code to prevent redundant dependency downloads).
  - [ ] **Test Execution Efficiency**: Are tests structured to execute efficiently? (e.g., leveraging `t.Parallel()` in Go unit tests where appropriate, avoiding redundant database migrations/re-seeding per test case by reusing Testcontainers instances).
  - [ ] **Asset & Dependency Scoping**: Are new dependencies strictly necessary? Ensure large static assets or unnecessary development files are excluded from production build contexts via `.dockerignore` / `.gitignore`.

### Persona 4: Experienced Reliability Engineer (SRE)
**Focus**: Observability, Meaningful Debugging Signals, and Monitoring Cost Efficiency.
- **Checklist**:
  - [ ] **Structured & Scoped Logging**: Are log messages structured and categorized at appropriate levels (`INFO`, `WARN`, `ERROR`, `DEBUG`)? Ensure high-frequency recurring events (e.g., SSE heartbeats, background polling ticks) use `DEBUG` level or are aggregated to prevent log spam.
  - [ ] **Meaningful Behavioral Metrics**: Are critical server behaviors (e.g., BGG sync batch completion rates, SSE connection drops, database query latency spikes) observable without logging redundant, low-value intermediate states?
  - [ ] **Telemetry Cost & Noise Hygiene**: Ensure logging statements do not emit large raw payloads, redundant stack traces for expected client errors, or PII/sensitive session tokens that inflate monitoring ingestion costs.

### Persona 5: Experienced Security Engineer (Full-Stack - Final Gate)
**Focus**: Vulnerability Minimization, Exploit Prevention, and Final Security Sign-Off.
- **Checklist (Backend & API)**:
  - [ ] **SQL Injection (SQLi)**: Are all database queries fully parameterized? Ensure zero raw dynamic SQL string concatenation exists in `internal/postgres/`.
  - [ ] **Authentication & Authorization (IDOR)**: Are explicit authorization checks enforced on all endpoint handlers? Ensure users cannot access or modify party/wishlist data belonging to other users.
  - [ ] **Information Disclosure**: Are internal server errors and stack traces properly masked before being returned in HTTP responses to the client? Ensure no new logging statements added by SRE/DevOps expose sensitive session tokens or PII.
- **Checklist (Frontend & UI)**:
  - [ ] **Cross-Site Scripting (XSS)**: Verify that Angular's built-in DOM sanitization is preserved. Audit any use of `bypassSecurityTrustHtml` or direct DOM manipulation introduced during UX polishing.
  - [ ] **Cross-Site Request Forgery (CSRF) & Session Security**: Ensure authentication tokens (`signinToken`) use secure cookie attributes (`HttpOnly`, `Secure`, `SameSite`).
  - [ ] **Access Control & Entropy**: Ensure invitation links and short codes utilize secure, unpredictable random generation to prevent brute-force enumeration.
