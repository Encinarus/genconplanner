# Managing Your Schedule & Time Blocks

Once you've starred your favorite events, your **Starred Events Dashboard** (`/starred/[year]`) becomes your primary workspace for organizing your schedule, resolving time conflicts, and establishing healthy convention boundaries.

---

## 1. Exploring the Starred Dashboard Views

The dashboard provides two distinct layouts to help you visualize and manage your selections:

### 📅 Calendar View (`/starred/[year]/calendar`)
Provides a traditional hourly grid breakdown across all four days of Gen Con (Thursday through Sunday).
- **Conflict Spotting**: Overlapping events are displayed side-by-side in the same time slot, making it instantly obvious where you have scheduling bottlenecks.
- **Tier Filtering**: Use the radio buttons at the top of the calendar to filter the display. You can choose to view `All` starred events, or isolate only your `Must Have` or `Very Interested` selections to see your core schedule structure.

### 📑 By Type List View (`/starred/[year]/by_type`)
Organizes your starred events into expandable accordion groups based on their game system or title.
- **Session Comparison**: Expand a group to see all available sessions side-by-side, comparing room locations and remaining ticket counts.
- **Group-Level Deletion**: Changed your mind about a game? Use the trashcan icon in the group header to remove all starred sessions of that game with a single click.

---

## 2. Decluttering Your Calendar: Toggling "Hide Backups"

When you star dozens of events across the `Somewhat Interested` tier, your calendar can quickly become crowded with overlapping boxes. 

```
┌────────────────────────────────────────────────────────┐
│ 📅 FullCalendar Toolbar                                │
│                                                        │
│  Filters: ( ) All  ( ) Must Have  ( ) Very Interested  │
│  Options: [☑] Hide Backups                             │
└────────────────────────────────────────────────────────┘
```

### The "Hide Backups" Toggle
Located directly inside the calendar toolbar, checking the **Hide Backups** box instantly hides all `Somewhat Interested` events from the grid. This allows you to inspect your primary schedule (`Must Have` and `Very Interested` events) for clean flow without permanently deleting your backup options. Uncheck the box whenever you want to see your backup options again.

---

## 3. Protecting Your Energy: Setting Time Blocks & Breaks

Gen Con is an exhausting four-day marathon. One of the most common mistakes attendees make is booking back-to-back events from 8:00 AM to midnight without leaving time to eat, rest, or travel between distant convention halls.

Gen Con Planner introduces powerful **Time Blocking** tools to protect your well-being.

```
┌────────────────────────────────────────────────────────┐
│ 🛡️ Custom Schedule Constraints                         │
│                                                        │
│  [Exclusive Block] ──► 12:00 PM - 1:30 PM (Lunch)      │
│  [Flexible Break]  ──► Min Duration: 45 mins between   │
└────────────────────────────────────────────────────────┘
```

### Exclusive Time Blocks
- **What is it?** A hard, non-negotiable lockout window (e.g., blocking out Friday from 5:00 PM to 7:00 PM for a family dinner).
- **System Behavior**: The Wishlist Engine will completely reject any event session that overlaps with an exclusive block, guaranteeing that time remains open on your final schedule.

### Flexible Breaks (Minimum Duration)
- **What is it?** A smart constraint that ensures you get a breather without forcing a rigid schedule lockout. You define a time window and a `min_duration_minutes` requirement (e.g., requesting at least a 45-minute break sometime between 11:30 AM and 2:30 PM).
- **System Behavior**: The optimization engine will evaluate various event permutations. It will allow you to book a morning event and an afternoon event, provided there is at least a 45-minute gap between them during your specified window.

---
> [!SUCCESS]
> Your schedule is now organized and protected! The final step before Gen Con registration opens is compiling your official export. Learn how the engine works in [Generating & Submitting Your Final Wishlist](./06-generating-final-wishlist.md).
