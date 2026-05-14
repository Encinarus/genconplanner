# Implementation Plan: Party Mode Phase 3

This phase introduces party-aware prioritization. Users can now specify modifiers that help the system understand *why* they are interested in an event, especially in the context of their party.

## User Review Required

> [!IMPORTANT]
> **Visibility of Modifiers**: These modifiers (`Better with others`, `Worth alone`, `Not worth alone`) will ONLY be visible and configurable if the user is currently a member of a party. For solo users, the UI will remain clean.

> [!IMPORTANT]
> **Category Defaults**: We will implement "User Category Overrides" which allow a user to say, for example, "I always prefer RPGs to be Better with others". This saves them from manually checking boxes for every event in that category.

## Proposed Changes

---

### [Component] Database & Schema

#### [MODIFY] [party_mode.sql](file:///Users/alek/projects/genconplanner/internal/postgres/party_mode.sql)
*   Add columns to `starred_events`:
    *   `better_with_others` BOOLEAN DEFAULT FALSE
    *   `worth_alone` BOOLEAN DEFAULT TRUE
    *   `not_worth_alone` BOOLEAN DEFAULT FALSE
*   [NEW] Create `user_category_preferences` table:
    ```sql
    CREATE TABLE public.user_category_preferences (
        email TEXT REFERENCES public.users(email),
        category TEXT, -- e.g., 'RPG', 'BGM'
        better_with_others BOOLEAN DEFAULT FALSE,
        worth_alone BOOLEAN DEFAULT TRUE,
        not_worth_alone BOOLEAN DEFAULT FALSE,
        PRIMARY KEY (email, category)
    );
    ```

---

### [Component] Backend (Go)

#### [MODIFY] [users.go](file:///Users/alek/projects/genconplanner/internal/postgres/users.go)
*   Update `StarredEvent` struct to include the three new boolean modifiers.
*   Modify `UpdateStarredEvent` and `updateStarredEventInternal` to accept and save these modifiers.
*   Update `fetchStarredInternal` to scan these columns.
*   [NEW] Implement `GetUserCategoryPreferences` and `UpdateUserCategoryPreference`.

#### [MODIFY] [prioritization.go](file:///Users/alek/projects/genconplanner/internal/prioritization/prioritization.go)
*   Enhance the scoring engine to be party-aware.
*   **Logic**:
    1.  Start with base tier score.
    2.  If `better_with_others` is TRUE: Add `N * 50` points where `N` is the number of other party members who have also starred this event (any tier).
    3.  If `not_worth_alone` is TRUE: If `N == 0`, subtract `500` points (effectively deprioritizing it).
    4.  If `worth_alone` is TRUE: Ensure base score is maintained regardless of `N`.

---

### [Component] Frontend (Angular)

#### [MODIFY] [api.service.ts](file:///Users/alek/projects/genconplanner/ui/src/app/services/api.service.ts)
*   Update `StarredEventDetail` interface.
*   Add methods for fetching/updating category preferences.

#### [MODIFY] [Starred Events Page]
*   Add a "Party Context" section to the star/tier selector (only if in a party).
*   Add checkboxes for:
    *   👥 Better with others
    *   👤 Worth doing alone
    *   🚫 Not worth doing alone

#### [NEW] [user-preferences.component.ts]
*   A new section in the User Profile to manage default preferences by category.

---

## Verification Plan

### Automated Tests
*   **Prioritization Tests**: Simulations in `internal/prioritization/prioritization_test.go` to verify score calculations.
*   **Database Tests**: Verify that modifiers are correctly persisted and retrieved.

### Manual Verification
1.  Join a party with a test account.
2.  Star an event and check "Better with others".
3.  Have another party member star the same event.
4.  Verify (via API or debug view) that the event's calculated score increases.
