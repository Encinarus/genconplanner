# Comprehensive Testing & Reliability Overhaul

This implementation plan outlines the architectural refactorings and new test infrastructure required to eliminate critical coverage gaps across the Gen Con Planner service. Specifically, it targets the PostgreSQL repository layer, background BGG/Gen Con synchronization workers, real-time Server-Sent Events (SSE) streaming, and core Angular frontend services/components.

## User Review Required

> [!IMPORTANT]
> **Docker Requirement for Integration Tests**: We are introducing `testcontainers-go` for database integration testing. Running `go test -tags=integration ./...` will require an active, local Docker daemon (e.g., Docker Desktop, Colima, or OrbStack) to spin up ephemeral PostgreSQL containers.

> [!WARNING]
> **Background Worker Refactoring**: Decoupling `internal/background` and `internal/bgg` from live HTTP calls and system time requires modifying function signatures to accept `BGGClient` and `Clock` interfaces. This is a structural change to how background workers are initialized in `cmd/update` and `cmd/web`.

## Open Questions

> [!NOTE]
> **Legacy Web Handlers (Resolved)**: Confirmed as intentionally deprecated in favor of the V2 Angular SPA (`v2.go`). These legacy server-side rendered routes will be excluded from the testing overhaul to focus resources entirely on the active API and SPA layers.

> [!NOTE]
> **Frontend E2E Framework (Resolved)**: Playwright has been selected as the official End-to-End testing framework for its seamless headless execution, cross-browser capabilities, and powerful network mocking.

---

## Proposed Changes

### Database & Repository Integration Testing (`internal/postgres` & `internal/testutil`)

To verify complex SQL queries, dynamic search filters, and PL/pgSQL triggers without brittle `sqlmock` strings, we will establish an ephemeral database testing suite using Testcontainers.

#### [NEW] [db.go](file:///Users/alek/projects/genconplanner/internal/testutil/db.go)
- Implement `SetupTestDB(t *testing.T) *sql.DB` using `testcontainers-go` to start an ephemeral `postgres:16-alpine` container.
- Configure automated execution of `internal/postgres/schema.sql` and custom SQL functions/triggers (`party_mode.sql`, `flexible_blocks.sql`, `cluster_key_update.sql`, `remove_org_trigger.sql`) upon container startup.

#### [NEW] [seed.sql](file:///Users/alek/projects/genconplanner/internal/testutil/seed.sql)
- Create a rich, deterministic set of SQL test fixtures representing a realistic production state.
- Include 3 sample users, 1 active party (`short_code: CODE2026`) with leader/member associations, multiple event categories (`BGM`, `RPG`), active/inactive event instances, and simulated ticket purchases.

#### [NEW] [integration_test.go](file:///Users/alek/projects/genconplanner/internal/postgres/integration_test.go)
- Implement comprehensive integration tests executing against the ephemeral Testcontainers database.
- Validate complex repository queries including `SearchEvents`, `LoadPartyMemberPurchases`, `LoadAgenda`, `LoadStarredEventClusters`, and `LoadEventGroupsForCategory` to ensure correct filtering, aggregation, and trigger behavior under a real Postgres query planner.

---

### Background Synchronization & BGG API Unit Testing (`internal/background`, `internal/bgg`)

Currently at 0% coverage, the background workers will be refactored to use dependency injection, enabling pure in-memory unit testing of batching, rate limiting, and error handling.

#### [MODIFY] [game.go](file:///Users/alek/projects/genconplanner/internal/bgg/game.go)
- Define a clean `BGGClient` interface:
  ```go
  type BGGClient interface {
      FetchGameData(ctx context.Context, ids []string) ([]*bgg.Game, error)
  }
  ```

#### [MODIFY] [bgg.go](file:///Users/alek/projects/genconplanner/internal/background/bgg.go)
- Refactor synchronization functions (e.g., `UpdateBGGData`) to accept `BGGClient` and a `Clock` interface (abstracting `time.Sleep` and `time.Tick`).
- Ensure rate limiting (10-second intervals) and batching (up to 20 IDs per request) utilize the injected clock and client.

#### [MODIFY] [update.go](file:///Users/alek/projects/genconplanner/internal/background/update.go)
- Update worker initialization logic to inject concrete implementations of `BGGClient` and system clock during production startup.

#### [NEW] [bgg_test.go](file:///Users/alek/projects/genconplanner/internal/background/bgg_test.go)
- Implement unit tests using an in-memory mock `BGGClient` and a mock clock.
- Verify correct batch slicing (exactly 20 IDs per batch), strict adherence to the 10-second rate limit window, robust error recovery on BGG API timeouts, and proper database caching behavior.

---

### Event Ingestion Unit Testing (`internal/events`)

Ensure the raw spreadsheet ingestion pipeline is resilient against malformed data and formatting variations.

#### [NEW] [ingestion_test.go](file:///Users/alek/projects/genconplanner/internal/events/ingestion_test.go)
- Implement unit tests for `xlsx.go` and `csv.go`.
- Use in-memory sample spreadsheet buffers to verify robust parsing of Gen Con event rows, correct handling of missing optional columns, accurate date/time conversion, and proper error surfacing for invalid file structures.

---

### Frontend Angular Services & Components (`ui/src/app`)

Expand Vitest coverage to protect real-time streaming, API communication, and core interactive components.

#### [NEW] [party-stream.service.spec.ts](file:///Users/alek/projects/genconplanner/ui/src/app/services/party-stream.service.spec.ts)
- Abstract the browser `EventSource` creation behind an injection token or factory.
- Inject a mock Subject in Vitest to simulate server SSE heartbeats, broadcast party interest updates, simulate network dropouts, and verify Page Visibility API pausing/resuming logic without an active backend.

#### [NEW] [api.service.spec.ts](file:///Users/alek/projects/genconplanner/ui/src/app/services/api.service.spec.ts)
- Implement unit tests utilizing Angular's `HttpTestingController`.
- Verify correct outbound HTTP headers, cookie attachment (`signinToken`), error interception, and data deserialization across REST endpoints.

#### [NEW] [party.service.spec.ts](file:///Users/alek/projects/genconplanner/ui/src/app/services/party.service.spec.ts)
- Implement unit tests verifying party creation, joining, renaming, member updating, and deletion API calls using `HttpTestingController`.

#### [NEW] [party.component.spec.ts](file:///Users/alek/projects/genconplanner/ui/src/app/components/party/party.component.spec.ts)
- Implement component test verifying UI interactions: party leader administrative controls, member renaming modals, short code invitation link generation, and reactive updates from `PartyStreamService`.

#### [NEW] [event-detail.component.spec.ts](file:///Users/alek/projects/genconplanner/ui/src/app/components/event-detail/event-detail.component.spec.ts)
- Implement component test verifying event metadata rendering, star button state toggling, and similar event loading.

---

### Frontend End-to-End Testing (`ui/e2e`)

Introduce Playwright to protect critical user workflows, utilizing network mocking for fast, isolated UI verification and full E2E execution against local backend/Testcontainers instances.

#### [NEW] [playwright.config.ts](file:///Users/alek/projects/genconplanner/ui/playwright.config.ts)
- Configure Playwright for cross-browser testing (Chromium, Firefox, WebKit).
- Define base URL (`http://localhost:4200`), timeout configurations, and web server startup commands for local E2E runs.

#### [NEW] [party-hub.spec.ts](file:///Users/alek/projects/genconplanner/ui/e2e/party-hub.spec.ts)
- Implement E2E test verifying Party Hub flows: creating a party, copying invite links, joining via short codes, and testing administrative actions (renaming, member management).
- Utilize Playwright's `page.route()` to mock API responses for fast UI validation, as well as full live backend verification.

#### [NEW] [wishlist.spec.ts](file:///Users/alek/projects/genconplanner/ui/e2e/wishlist.spec.ts)
- Implement E2E test verifying the wishlist prioritization engine: starring events, toggling interest tiers (Must Have, Very Interested), and verifying that flexible/exclusive blocked times correctly update the schedule UI.

---

## Verification Plan

### Automated Tests

#### Backend Unit Tests
Execute standard Go unit tests to verify isolated background worker, BGG client, and event ingestion logic:
```bash
go test -v ./internal/background/... ./internal/bgg/... ./internal/events/...
```

#### Backend Integration Tests
Execute Testcontainers-backed repository integration tests (requires active Docker daemon):
```bash
go test -tags=integration -v ./internal/postgres/... ./internal/api/...
```

#### Frontend Unit & Component Tests
Execute Vitest test suite to verify Angular services, SSE streaming, and UI components:
```bash
cd ui && npm test -- --watch=false
```

#### Frontend End-to-End Tests
Execute Playwright test suite to verify full user journeys and UI workflows:
```bash
cd ui && npx playwright test
```

### Manual Verification

#### 1. Docker Container Lifecycle Validation
- During `go test -tags=integration ./...`, observe Docker Desktop / CLI (`docker ps`) to verify that the ephemeral `postgres:16-alpine` container spins up, executes migrations/seeds, runs tests cleanly, and is automatically destroyed upon test completion.

#### 2. Local End-to-End Workflow Parity
- Stand up the local development environment (`docker-compose up` or `npm run start` / `go run cmd/web/main.go`).
- Verify core user journeys against a locally seeded database:
  1. Log in and navigate to the Party Hub.
  2. Create a new party, generate an invitation link, and join as a second user.
  3. Star an event and observe real-time interest updates propagating across party members via SSE.
