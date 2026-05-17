# Gen Con Planner Chrome Extension

This browser extension enhances event detail pages on [gencon.com](https://www.gencon.com) (e.g., `https://www.gencon.com/events/<number>`) and provides seamless integration with [genconplanner.com](https://www.genconplanner.com).

## Features

- **Ticket Purchase Section Organization**: Automatically organizes the ticket selection list on Gen Con event pages. It places a divider (`<hr>`) after your personal ticket options ("Myself" and "Another ticket for me") and sorts all remaining friend/group ticket options alphabetically by their name.

## Installation (Developer Mode)

1. Open Google Chrome and navigate to `chrome://extensions/`.
2. Enable **Developer mode** using the toggle switch in the top right corner.
3. Click the **Load unpacked** button in the top left corner.
4. Select the `extension` directory located in your `genconplanner` project folder (`/Users/alek/projects/genconplanner/extension`).
5. The extension is now active! Visit any Gen Con event detail page to see it in action.

## Structure

- `manifest.json`: Manifest V3 configuration defining extension metadata, permissions, and content script matching rules.
- `content_gencon.js`: The content script responsible for observing the DOM, isolating personal ticket options, inserting the dividing `<hr>`, and alphabetically sorting the remaining ticket options.
