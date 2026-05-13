# Requirements Specification: Party Mode

This document outlines the features and logic for the "Party Mode" in Gen Con Planner.

## 1. Core Concepts & Definitions
*   **Party**: A group of users planning together for a specific Gen Con year.
*   **Member**: A user who has joined a party.
*   **Party Leader**: The creator of the party (or transferee). Has administrative privileges.
*   **Ticket**: A specific entry for an event instance. Can be "Purchased" by one user and "Claimed" by another.
*   **Claim**: Represents the person actually attending the event using a specific ticket.
*   **Transfer**: Moving the "cost responsibility" or "permanent possession" of a ticket to another user.

## 2. Party Lifecycle & Roles

### 2.1 Creation & Leadership
*   **One per Year**: A user can create/be in only one party per Gen Con year.
*   **Party Leader Role**: The creator is the initial Party Leader.
*   **Party Leader Privileges**:
    *   Generate and manage invite links.
    *   Rename the party.
    *   Transfer leadership to another member.
    *   Manage **External Links**: (Google Docs, Google Spreadsheets, Splitwise).
        *   *Restriction*: Only URLs from trusted domains (e.g., `docs.google.com`, `splitwise.com`) are allowed.
*   **Leaving**: 
    *   The Party Leader cannot leave unless they transfer leadership OR they are the last remaining member.
*   **Deletion**: 
    *   A party can only be deleted if it has exactly one member (the Party Leader) and no tickets have been transferred into the group from outside (or specifically from members who left).

## 3. Ticket & Claim Management

### 3.1 The Claim System
*   **Purchaser vs. Recipient**: Tracks who bought the ticket and whose name is on it in the Gen Con system.
*   **Default State**: A ticket is initially "claimed" by the person whose name was on the purchase.
*   **Releasing Claims**: Any member can release their claim on a ticket, making it "unclaimed."
*   **Claiming**: Any member in the party can claim an "unclaimed" ticket.
*   **Ticket Types**: The system must track if a ticket is **Digital** or **Physical**.

### 3.2 Financial Tracking
*   **Cost Logging**: The system tracks the purchase price of each ticket.
*   **Debt Logic**: If User A purchases a ticket and User B claims it, the system records that User B owes User A the cost of that ticket.
*   **Accounting View**: A summary page/section showing who owes whom and for which tickets.

### 3.3 Conflict & Opportunity Analysis
*   **Overlap Highlighting**: 
    *   The UI should highlight conflicts (multiple tickets in the same time slot).
    *   *Policy*: Conflicts are **allowed** (for flexibility), but clearly flagged.
*   **Opportunity Highlighting**: When User A views unclaimed tickets, the system highlights those for events that **do not overlap** with User A's current claims.

## 4. User Views & Profile

### 4.1 Agenda View (Personal Profile)
*   **Daily Schedule**: A list of all *claimed* events for each day.
*   **Sorting**: Primary sort by Start Time, secondary by End Time (for conflicts).
*   **Details**: Event Name, Location, Ticket Type (Physical/Digital), and a list of **other party members** attending the same slot.
*   **Unclaimed Inventory**: A list of tickets purchased by the user that are currently unclaimed (useful for returns).
*   **Transfer History**: Ability to see the last person a ticket was transferred to.

## 5. Group Persistence & Leaving Logic

### 5.1 Fate of Tickets upon Leaving
When User A leaves the party, the system must handle their tickets based on their status:
*   **Claimed (Not Purchased/Transferred)**: The claim is removed, and the ticket returns to "unclaimed" status within the party.
*   **Transferred (Purchased by others)**: The ticket stays with User A. They are still responsible for the cost to the original purchaser.
*   **Purchased (Not Transferred)**: These tickets are **removed** from the party. Any other members who had claims on them lose those claims. User A sees these as "unclaimed" on their personal profile.
*   **Purchased (Transferred)**: These tickets stay with the transferee in the party. The system continues to track that User A was the original purchaser.

*Note: The UI should ideally prompt/force the user to confirm the fate of each relevant ticket before they are allowed to leave.*

## 6. Collaborative Planning

### 6.1 Interest Tiers & Prioritization (Personal)
Users can rank their interest for each event grouping. This data drives the personal wishlist and the group prioritization.
*   **Tiers**: `Must Have`, `Very Interested`, `Somewhat Interested`.
*   **Tie-Breaker Modifiers**:
    *   `Better with others`: Prioritize if other party members are also interested.
    *   `Worth doing alone`: Prioritize even if no one else joins.
    *   `Not worth doing alone`: **New**: If no one else in the party claims/buys this, deprioritize or remove from wishlist.
    *   *Note*: These modifiers are **only visible/available** if the user is currently in a party. For solo users, these are hidden.
    *   *Note*: These are not mutually exclusive (e.g., `Better with others` + `Worth doing alone`).
*   **Default Behavior Hierarchy**:
    1.  **System Defaults**: Pre-configured defaults per category (e.g., RPGs might be "Better with others", Solo games "Worth alone").
    2.  **User Category Overrides**: User can set their own default preference per category from their profile.
    3.  **User Event Overrides**: User can manually override specific event groups.

### 6.2 Wishlist Prioritization Engine
Gen Con wishlists are limited to **50 events**. The system helps optimize this list.

*   **Ranking Logic**:
    1.  Primary: Personal Interest Tier.
    2.  Secondary: Tie-breakers (Others interested + Modifiers).
*   **Anti-Spam Optimization**: The system will not fill a wishlist with too many instances of the same event group (e.g., 50 instances of the same escape room). It will select the few instances that maximize the number of interested party members available.
*   **Altruistic Filling**: If a user has < 50 events, they can enable "Fill for Others."
    *   The system picks `Must Have` events for other party members that fit into the user's empty time slots.
    *   *UI Requirement*: These must be clearly labeled as "In wishlist for [Member Name]".
*   **Purchase Awareness**:
    *   As soon as a ticket is marked as **Purchased** (for a specific person/event pair), that pair is removed from wishlist prioritization.
    *   The system will explicitly highlight that the user should *not* purchase an event if a ticket has already been secured for them by someone else.
*   **Asynchronous Processing**:
    *   Optimization calculations do not need to be synchronous with every star/priority change.
    *   The system can batch-process these updates in the background.
    *   *UI Requirement*: Show the "Last Optimized At" timestamp to the user.
*   **Solo Optimization (Party-of-One)**:
    *   The prioritization logic applies even to users not in a party. 
    *   They are treated as a "party of size one" for scoring purposes.
    *   The engine first calculates personal rank, then adjusts based on the party context (if any).

### 6.3 Gen Con Site Integration (Manual)
*   **Submission Tracking**: Users can mark their wishlist as "Assembled/Submitted" on the Gen Con site.
*   **State Tracking**: The system tracks the timestamp of submission.

## 7. Event Booking Day (The "War Room")
A high-intensity coordination mode for the day Gen Con processes wishlists.

### 7.1 Pre-Game Phase: Lock-In
*   **Trigger**: The Party Leader can "Start Event Day" mode.
*   **Wishlist Snapshots**: Before processing begins, every member must "Lock In" their final wishlist. This creates a snapshot of what was actually submitted to Gen Con.
*   **Leader Dashboard**: The leader can see a status list of which party members have locked in their wishlists and who is still pending.

### 7.2 Execution Phase: Reservation Tracking
As Gen Con processes wishlists, members update their status in real-time:
*   **Lock Status**: Members mark items in their locked-in wishlist as `Reserved`, `Failed` (unavailable), or `Pending`.
*   **Expiration Timer**: For `Reserved` items, members record the **Expiration Time** (Gen Con typically gives a 2-hour window to purchase).
*   **Redundancy Check**: The system highlights if multiple members have locks on the same event cluster or overlapping sessions.

### 7.3 Decision Support & Purchase Coordination
*   **Group Status Dashboard**: A real-time view for all members showing:
    *   Total Tickets Needed for a specific event cluster.
    *   Total Tickets Currently Locked (and by whom).
    *   Total Tickets Actually Purchased.
    *   Gap Analysis: Who still needs a ticket but doesn't have a lock/purchase.
*   **Purchase Confirmation**: As members confirm purchases, the dashboard updates to show which "locks" are no longer needed and can be allowed to expire or be released.

## 8. Phased Rollout Plan

### Phase 1: Basic Party Management & Visibility
*   **Core Party Lifecycle**: Create, Join (via link), Leave.
*   **Leadership Management**: Transfer leadership to another member, Deletion/Closing of party (under specific conditions).
*   **Party Details**: Rename party, manage basic info.
*   **Shared Visibility**: Shared member list and visibility of each other's starred events.

### Phase 2: Personal Interest Tiers & Solo Optimization
*   Implementing the `Must Have` / `Interested` tiers.
*   Background prioritization engine for "Party of One."
*   Agenda View on the profile (based on stars initially).

### Phase 3: Party-Aware Personal Prioritization
*   Adding tie-breaker modifiers (`Better with others`, `Worth alone`, etc.) - visible only to party members.
*   Scoring adjustments based on the interest of other party members.
*   System/User defaults for these modifiers.

### Phase 4: Collaborative Group Prioritization
*   The "Wishlist Prioritization" tab in the Party view.
*   Algorithm refinements: Anti-spam (capping instances) and Altruistic filling.

### Phase 5: Ticket Management & Claims
*   The "Purchased" vs. "Claimed" tracking system.
*   Financial tracking (Cost logging and debt calculation).
*   Physical vs. Digital ticket tracking.

### Phase 6: Advanced Analysis & Leaving Logic
*   Overlap highlighting and Opportunity analysis.
*   Implementing the complex fate logic when a member leaves.

### Phase 7: Event Booking Day ("War Room")
*   The full coordination flow: Lock-ins, real-time reservation tracking, and the purchase dashboard.

## 9. Progress Tracking

### Status Summary
- **Phase 1: Basic Party Management & Visibility** - [x] **Completed**
- **Phase 2: Personal Interest Tiers & Solo Optimization** - [x] **Completed**
- **Phase 3: Party-Aware Personal Prioritization** - [ ] **Remaining**
- **Phase 4: Collaborative Group Prioritization** - [ ] **Remaining**
- **Phase 5: Ticket Management & Claims** - [ ] **Remaining**
- **Phase 6: Advanced Analysis & Leaving Logic** - [ ] **Remaining**
- **Phase 7: Event Booking Day ("War Room")** - [ ] **Remaining**

### Implementation Notes
- **Phase 1**: Core backend logic implemented in `internal/postgres/party.go`. V2 UI component implemented in `ui/src/app/components/party/`. Database schema updated with `parties` and `party_members` tables.
- **Phase 2**: Introduced `interest_tier` ENUM and `tier` column in `starred_events`. Updated API to support tiered starring. Implemented Agenda View in `AgendaComponent` and integrated into the user profile.
