# Implementation Plan: Party Mode Phase 2

This phase introduces granular interest tiers (`Must Have`, `Very Interested`, `Somewhat Interested`) and the foundation for the personal Agenda view.

## User Review Required

> [!IMPORTANT]
> **Interest Tier Data Type**: We will use a Postgres ENUM for `interest_tier` (`must_have`, `very_interested`, `somewhat_interested`) to ensure data integrity.

> [!IMPORTANT]
> **UI Interaction Flow**: 
> - **Event Cards/Details**: The "Star" button remains a simple binary toggle. When starring, it defaults to `very_interested`.
> - **My Starred Events**: Tier selection (radio buttons) will be added to this page to allow granular ranking after starring.

## Proposed Changes

---

### [Component] Database & Schema

#### [MODIFY] [party_mode.sql](file:///Users/alek/projects/genconplanner/internal/postgres/party_mode.sql)
*   Define the ENUM type: `CREATE TYPE interest_tier AS ENUM ('must_have', 'very_interested', 'somewhat_interested');`
*   Add the `tier` column to `starred_events`: `ALTER TABLE starred_events ADD COLUMN tier interest_tier DEFAULT 'very_interested';`

---

### [Component] Backend (Go)

#### [MODIFY] [users.go](file:///Users/alek/projects/genconplanner/internal/postgres/users.go)
*   Update `StarredEvent` struct to include `Tier string`.
*   Modify `UpdateStarredEvent` to accept `tier` as a parameter.
*   Modify `fetchStarredInternal` to scan the `interest_tier` column.
*   Update `BulkStarEvents` to handle tiers if provided.

#### [NEW] [agenda.go](file:///Users/alek/projects/genconplanner/internal/postgres/agenda.go)
*   Implement `LoadAgenda(db *sql.DB, userEmail string, year int)` which fetches starred events and sorts them by time for the Agenda view.

#### [MODIFY] [api.go](file:///Users/alek/projects/genconplanner/internal/web/api.go) (Assuming this is where endpoints are)
*   Update `/api/v1/user/star` to accept a `tier` in the request body.
*   Add `/api/v1/user/agenda/{year}` endpoint.

---

### [Component] Frontend (Angular)

#### [MODIFY] [api.service.ts](file:///Users/alek/projects/genconplanner/ui/src/app/services/api.service.ts)
*   Update `StarredEventDetail` interface to include `tier`.
*   Update `starEvent()` method signature to include `tier`.
*   Add `getAgenda(year: number)` method.

#### [MODIFY] [Event Card / Detail Components]
*   Keep the "Star" button as a binary toggle.
*   When adding a star, default to `very_interested`.

#### [MODIFY] [Starred Events Page]
*   Add a radio button group (or segmented control) for each starred event to change its `tier`.
*   Tiers: 🔥 `Must Have`, ⭐ `Very Interested`, 👍 `Somewhat Interested`.

#### [NEW] [agenda.component.ts](file:///Users/alek/projects/genconplanner/ui/src/app/components/agenda/agenda.component.ts)
*   Create a simple chronological list view of starred events for the profile page.

---

### [Component] Prioritization Engine (Solo)

#### [NEW] [prioritization.go](file:///Users/alek/projects/genconplanner/internal/prioritization/prioritization.go)
*   Implement a basic scoring engine:
    *   `Must Have` = 100 points
    *   `Very Interested` = 50 points
    *   `Somewhat Interested` = 10 points
*   This will be used to sort the "Wishlist" in future phases but can be introduced now for the Starred page sorting.

## Verification Plan

### Automated Tests
*   **Unit Tests**: Add tests to `internal/postgres/users_test.go` for tiered starring logic.
*   **API Tests**: Verify that `/api/v1/user/star` correctly saves and returns the tier.

### Manual Verification
1.  Star an event as `Must Have` in the UI.
2.  Refresh the page and verify the tier is persisted.
3.  Navigate to the "Agenda" view and verify events are listed chronologically.
4.  Verify that legacy "starred" events still appear (with the default tier).
