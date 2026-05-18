# Collaborative Wishlist Building

When you join a party in Gen Con Planner, your individual planning efforts instantly become part of a larger collaborative ecosystem. The **Party Hub** serves as your group's command center, aggregating everyone's starred events and interest tiers into a unified consensus view.

---

## 1. The Party Hub & Shared Interests Aggregation

Navigating to your Party Hub (`/party/[id]`) reveals a dense, multi-tabbed coordination interface featuring an ultra-compact vertical navigation rail (`Events`, `Members`, `Wishlist`, `Settings`).

```
┌────────────────────────────────────────────────────────┐
│ 👥 Shared Interests Consensus View                     │
│                                                        │
│  🎲 Battlestar Galactica (Fantasy Flight Games)        │
│  [🔴 Must Have: Alek, Sarah]  [🟡 Interested: Dave]    │
│  Total Interested: 3/4 Members ──► [🔥 High Consensus] │
└────────────────────────────────────────────────────────┘
```

### How Consensus is Calculated
The platform groups identical event sessions (`cluster_id`) and aggregates every member's interest tier (`Must Have`, `Very Interested`, `Somewhat Interested`). 
- **Live SSE Updates**: Powered by Server-Sent Events (SSE), any time a party member stars an event or changes a priority tier on their personal screen, the update broadcasts instantly across all active client screens without requiring a page refresh. You can watch your group's consensus evolve in real-time during planning calls!

---

## 2. Tie-Breaker Modifiers for Group Planning

When planning solo, your wishlist order is determined strictly by your interest tiers. In Party Mode, Gen Con Planner unlocks advanced **Tie-Breaker Modifiers** to help the optimization engine prioritize events that match your group's social preferences.

```
┌────────────────────────────────────────────────────────┐
│ ⚙️ Party Tie-Breaker Modifiers                         │
│                                                        │
│  [Better with others]    ──► Boosts if friends join    │
│  [Worth doing alone]     ──► Keeps even if solo        │
│  [Not worth doing alone] ──► Drops if no friends join  │
└────────────────────────────────────────────────────────┘
```

### Applying Modifiers to Event Clusters
When viewing an event cluster within the Party Hub, you can toggle these modifiers to fine-tune how the engine treats the game:

- **`Better with others`**: Instructs the engine to actively boost the priority score of this event if multiple party members have also starred it. The more friends interested, the higher it climbs on your wishlist queue.
- **`Worth doing alone`**: Protects the event on your wishlist. Even if none of your fellow party members express interest, the engine guarantees this item retains its priority ranking.
- **`Not worth doing alone`**: A powerful filtering guardrail. If you tag an event with this modifier and no other party members star or claim a ticket for it, the engine will automatically deprioritize or strip the event from your final 50-item wishlist export to prevent wasting valuable slots on a game you don't want to play solo.

---

## 3. Group Wishlist Optimization & Altruistic Filling

When your Party Leader prepares the group's booking strategy, the **Wishlist Prioritization Tab** in the Party Hub provides an optimized master schedule for the entire table.

### Smart Group Capping (Anti-Spam)
Just like personal wishlist generation, the group engine prevents wishlist spam. If your party has 5 members who all want to do a specific escape room, the engine won't flood your collective wishlists with 50 different sessions of that room. It identifies the 2 or 3 specific time slots where all 5 members are free and concentrates your group's bidding power on those exact sessions.

### Altruistic "Fill for Others"
If Member A has only 30 events on their personal wishlist, the engine can automatically utilize their remaining 20 slots to bid on `Must Have` events for Member B and Member C. This dramatically increases your group's overall chances of securing high-demand tickets during Gen Con's registration lottery!

---
> [!SUCCESS]
> Your group's wishlists are now perfectly aligned! Once tickets are secured, you need to track who holds which pass and settle up costs. Learn how in [Managing Tickets, Claims & Financial Tracking](./09-managing-tickets-claims-finances.md).
