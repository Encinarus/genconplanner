# Implementation Plan: Party Mode Phase 2.5 (Revised)

This phase implements the "Personal Wishlist Optimization" engine. It generates a 50-item wishlist that prioritizes user intent (Hearts vs Stars), manages scarcity (rare tickets first), and optimizes for a packed calendar.

## User Review Required

> [!IMPORTANT]
> **Scarcity Weighting**: We will prioritize events with fewer total tickets or sessions. This means a `Must Have` event with only 4 tickets will likely be #1 on your wishlist, even if you have other `Must Have` events.

> [!IMPORTANT]
> **Packing Heuristic**: The algorithm will try to select sessions that don't overlap. If Event A is available at two times, and one of those times is the *only* time Event B is available, the system will shift Event A to its other slot to fit both.

> [!IMPORTANT]
> **Anti-Spam Threshold**: We will enforce a **strict limit of 3 sessions per event group** in the final wishlist. Even if you heart every session of "True Dungeon," the system will select the 3 that maximize your overall schedule and put them in priority order.

## Proposed Changes

---

### [Component] Backend (Go)

#### [MODIFY] [prioritization.go](file:///Users/alek/projects/genconplanner/internal/prioritization/prioritization.go)
*   Implement the multi-factor scoring model:
    *   **Tier Points**: `Must Have` (10,000), `Very Interested` (1,000), `Somewhat Interested` (100).
    *   **Scarcity Bonus**: Based on total group tickets and number of sessions.
*   Implement the **Two-Pass Greedy Selection**:
    1.  **Pass 1 (Perfect Calendar)**: Select the highest-priority session for each group that has zero conflicts with previously selected sessions.
        - **Group Ordering**: Tier (Primary) > Scarcity (Secondary) > Availability (Tertiary).
    2.  **Pass 2 (Backups)**: Fill remaining slots (up to 50 total) with the next-best sessions for each group, capped at **3 total sessions per group**. These are ordered by priority but allowed to conflict with the "Perfect Calendar" sessions.
*   **Wishlist Sequencing**: Order the final list with "Perfect Calendar" items first, then "Backups".

#### [MODIFY] [api.go](file:///Users/alek/projects/genconplanner/internal/web/api.go)
*   Add `/api/v1/user/wishlist/{year}` endpoint.

---

### [Component] Frontend (Angular)

#### [MODIFY] [Starred Events Page]
*   Add a new **"Wishlist" tab** between the "By Type" and "Bulk" tabs.
*   This view will display the prioritized 50-item wishlist.
*   Visual Indicators for `Primary` vs `Backup` items.
*   Badges for `Rare`, `Must Have`, etc.

---

## Verification Plan

### Automated Tests
*   **Conflict Resolution Test**: Ensure rare events win contested slots.
*   **Packing Test**: Ensure events shift to available slots to accommodate single-slot events.

### Manual Verification
1. Star 60+ events.
2. Verify the wishlist view generates a logical 50-item list.
