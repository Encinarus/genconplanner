# Implementation Plan: Party Mode Phase 4

This phase transitions the Party view into a dense, full-page collaborative planning hub. It introduces an ultra-compact icon navigation rail, a Master-Detail split pane for exploring shared event interests, and Server-Sent Events (SSE) for instantaneous, real-time synchronization across party members.

## User Review Required

> [!IMPORTANT]
> **Real-Time Push Architecture (SSE)**: We will implement Server-Sent Events (`text/event-stream`) backed by Postgres `LISTEN/NOTIFY`. When any member updates their interest tier, active party screens will update instantaneously without HTTP polling.

> [!IMPORTANT]
> **Master-Detail UI Paradigm**: To replace the legacy spreadsheet workflow, the `Events` tab will feature a scannable master list of event group cards on the left and a rich drilldown detail panel on the right. The drilldown panel will maximize code reuse by embedding the top half of the standalone Event Details component (omitting sub-instance clutter).

## Proposed Changes

---

### [Component] Database & Backend Pub/Sub

#### [MODIFY] [party_mode.sql](file:///Users/alek/projects/genconplanner/internal/postgres/party_mode.sql)
*   Add Postgres trigger and function to broadcast notifications on `starred_events` table changes:
    ```sql
    CREATE OR REPLACE FUNCTION public.notify_party_interest_update()
    RETURNS TRIGGER AS $$
    DECLARE
        v_party_id INT;
    BEGIN
        -- Find active parties for the user
        FOR v_party_id IN 
            SELECT party_id FROM public.party_members WHERE email = COALESCE(NEW.email, OLD.email)
        LOOP
            PERFORM pg_notify('party_updates', json_build_object(
                'party_id', v_party_id,
                'event_id', COALESCE(NEW.event_id, OLD.event_id),
                'email', COALESCE(NEW.email, OLD.email),
                'tier', NEW.tier
            )::text);
        END LOOP;
        RETURN NEW;
    END;
    $$ LANGUAGE plpgsql;

    CREATE OR REPLACE TRIGGER trig_party_interest_update
    AFTER INSERT OR UPDATE OR DELETE ON public.starred_events
    FOR EACH ROW EXECUTE FUNCTION public.notify_party_interest_update();
    ```

---

### [Component] Backend (Go)

#### [MODIFY] [party.go](file:///Users/alek/projects/genconplanner/internal/postgres/party.go)
*   Implement `LoadPartySharedInterests(db *sql.DB, partyId int64, year int) ([]*SharedInterestGroup, error)`:
    *   Joins `parties`, `party_members`, `starred_events`, `events`, and `users`.
    *   Groups by `cluster_id` to aggregate total sessions, total tickets, and a JSON array of member interest tiers (`Must Have`, `Very Interested`, `Somewhat Interested`).

#### [NEW] [pubsub.go](file:///Users/alek/projects/genconplanner/internal/pubsub/pubsub.go)
*   Implement a broadcast hub managing `pq.Listener` on the `party_updates` channel.
*   Maintain active client subscriptions grouped by `party_id`.

#### [MODIFY] [api.go / web handlers]
*   Add GET `/api/v1/parties/{id}/interests` endpoint returning aggregated `SharedInterestGroup` DTOs.
*   Add GET `/api/v1/parties/{id}/stream` endpoint implementing Gin's `c.Stream()` for Server-Sent Events.

---

### [Component] Frontend (Angular)

#### [MODIFY] [api.service.ts](file:///Users/alek/projects/genconplanner/ui/src/app/services/api.service.ts)
*   Add `SharedInterestGroup` and `MemberInterest` interfaces.
*   Add `getPartyInterests(partyId: number, year: number)` method.

#### [NEW] [party-stream.service.ts](file:///Users/alek/projects/genconplanner/ui/src/app/services/party-stream.service.ts)
*   Manage native `EventSource` connection to `/api/v1/parties/{id}/stream`.
*   Expose Angular Signals (e.g., `latestInterestUpdate`) for surgical, real-time DOM updates.

#### [MODIFY] [party.component.ts & party.component.html]
*   Refactor into a full-page master layout with persistent header and ultra-compact, icon-only vertical navigation rail (`Events`, `Members`, `Calendar`, `Settings`).

#### [NEW] [party-interests.component.ts & party-interests.component.html]
*   **Master List (Left)**: Renders event group cards featuring inline compact radio buttons (`❤️`, `⭐`, `👍`) and member avatar clusters.
*   **Drilldown Panel (Right)**: Reuses the top half of the standalone Event Details component and displays a prominent roster of interested party members with collaborative notes.

---

## Verification Plan

### Automated Tests
*   **Backend Unit Tests**: Add tests in `internal/postgres/party_test.go` verifying `LoadPartySharedInterests` correctly groups by `cluster_id` and aggregates member tiers.
*   **Pub/Sub Tests**: Verify `pubsub.go` correctly routes broadcast messages to the appropriate `party_id` subscribers.

### Manual Verification
1.  Open Party A in two separate browser windows (Window 1: User Alice, Window 2: User Bob).
2.  In Window 1, Alice clicks ❤️ on "True Dungeon: Odin's Haven".
3.  Verify Window 2 (Bob's screen) instantaneously displays Alice's avatar with a ❤️ badge on the True Dungeon card without a page refresh.
4.  Click the True Dungeon card in Window 2 and verify the right drilldown panel displays the correct top-half event details and member roster.
