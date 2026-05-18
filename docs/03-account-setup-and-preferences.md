# Account Setup & Profile Preferences

While Gen Con Planner allows anyone to browse the event catalog and search for games without logging in, creating a free account unlocks the platform's true power: personal wishlist curation, schedule protection, and collaborative Party Mode planning.

This guide walks you through setting up your account, securing your profile, and configuring your baseline scheduling preferences.

---

## 1. Signing In with Google

Gen Con Planner utilizes Firebase Authentication for secure, passwordless login via your existing Google Account. This ensures your data is safely backed up in the cloud and synchronized across all your devices (desktop, laptop, and mobile).

### Step-by-Step Login
1. Locate the **Sign In** button in the top right corner of the navigation bar.
2. Click **Sign in with Google**.
3. A secure popup window will appear. Select your preferred Google account.
4. Once authenticated, the navigation bar will update to display your profile avatar and give you access to your personal Starred Events (`/starred`) and Party Hub (`/party`).

---

## 2. Configuring Your User Profile

After logging in for the first time, click on your profile avatar or navigate to `/user` to access your **User Profile Settings**.

### Essential Profile Fields
- **Display Name**: Set a recognizable name. This is how you will appear to friends when joining a party, locking in wishlists, or claiming tickets.
- **Email Address**: Displayed for account verification and group invitation management.
- **Personal Agenda View**: A quick snapshot of your currently claimed and purchased tickets for the upcoming convention.

---

## 3. Setting Category Default Interest Overrides

One of Gen Con Planner's most powerful automation features is the ability to establish **Category Default Preferences**. When you are part of a planning party, these preferences automatically inform the group's wishlist prioritization engine without requiring you to manually configure every single event.

```
┌────────────────────────────────────────────────────────┐
│ ⚙️ Category Preference Overrides                       │
│                                                        │
│ Role Playing Games (RPG) ──► [Better with others]      │
│ Board Games (BGM)        ──► [Worth doing alone]       │
│ Supplemental (SUP)       ──► [Not worth doing alone]   │
└────────────────────────────────────────────────────────┘
```

### Understanding Preference Modifiers
Within your profile settings, you can assign default behavior modifiers to any major event category:

- **`Better with others`**: Signals to the prioritization engine that you prefer attending these events with fellow party members. The system will actively boost event sessions where multiple friends are available. *(Perfect for team-based RPGs or cooperative board games).*
- **`Worth doing alone`**: Indicates that you are highly passionate about this category and want it prioritized on your wishlist even if no one else in your party joins you. *(Great for miniature painting workshops or niche strategy games).*
- **`Not worth doing alone`**: Specifies that you only want to attend these events if at least one other party member secures a ticket alongside you. If no friends join, the system will automatically deprioritize or drop these events from your final wishlist to save valuable slots. *(Ideal for escape rooms or escape-style puzzle games).*

### How Overrides Work
1. **System Defaults**: Out of the box, Gen Con Planner assigns sensible baseline defaults (e.g., RPGs default to "Better with others").
2. **Your Category Overrides**: Setting a preference in your User Profile overrides the system default for all events in that category.
3. **Individual Event Overrides**: When viewing a specific event cluster, you can always manually override your category preference for that specific game.

---
> [!SUCCESS]
> Your account is now fully configured! You are ready to start building your custom convention schedule. Head over to [Curating Your Wishlist with Interest Tiers](./04-mastering-interest-tiers.md) to master the starring system.
