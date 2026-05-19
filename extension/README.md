# Gen Con Planner Chrome Extension

This browser extension enhances event detail pages and transaction history on [gencon.com](https://www.gencon.com) and provides seamless integration with [genconplanner.com](https://www.genconplanner.com).

## Features

- **Ticket Purchase Section Organization**: Automatically organizes the ticket selection list on Gen Con event pages. It places a divider (`<hr>`) after your personal ticket options ("Myself" and "Another ticket for me") and sorts all remaining friend/group ticket options alphabetically by their name.
- **Automated Ticket Synchronization**: Detects ticket transactions on your Gen Con Transactions page (`/profile/transactions`) for the current year. Displays a premium interactive banner allowing you to instantly sync your purchased tickets directly to your active party in Gen Con Planner.

## Configuration & Deployment

By default, the extension is configured for local development and communicates with `http://localhost:8080`. 

**For Production Deployment**:
Before publishing or deploying the extension for production use, open `background.js` and update `SERVER_URL` to the live domain:
```javascript
const SERVER_URL = 'https://genconplanner.com';
```

## Installation (Developer Mode)

1. Open Google Chrome and navigate to `chrome://extensions/`.
2. Enable **Developer mode** using the toggle switch in the top right corner.
3. Click the **Load unpacked** button in the top left corner.
4. Select the `extension` directory located in your `genconplanner` project folder (`/Users/alek/projects/genconplanner/extension`).
5. The extension is now active! Visit any Gen Con event detail page or your transactions page to see it in action.

## Structure

- `manifest.json`: Manifest V3 configuration defining extension metadata, permissions (`cookies`, `storage`), host permissions, and content script matching rules.
- `background.js`: Background service worker responsible for securely retrieving the user's Gen Con Planner auth token (`signinToken` cookie) and handling cross-origin API sync requests.
- `content_gencon.js`: Content script responsible for DOM observation, ticket purchase UI organization, transaction history parsing, and rendering the interactive sync banner.
