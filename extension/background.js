/**
 * Gen Con Planner Extension - Background Service Worker
 * 
 * Handles cross-origin requests to the Gen Con Planner server API
 * by retrieving the user's signinToken cookie and making the POST request.
 */

// CONFIGURATION: Set to http://localhost:8080 for local development.
// For production deployment, update to https://www.genconplanner.com
const SERVER_URL = 'http://localhost:8080';

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'sync_tickets') {
    // Retrieve the signinToken cookie for the configured server URL
    chrome.cookies.get({ url: SERVER_URL, name: 'signinToken' }, (cookie) => {
      if (!cookie) {
        sendResponse({ 
          status: 'error', 
          message: `Not logged into Gen Con Planner. Please log in at ${SERVER_URL} first.` 
        });
        return;
      }

      const currentYear = new Date().getFullYear();
      const endpoint = `${SERVER_URL}/api/v1/party/${currentYear}/tickets/sync`;

      fetch(endpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${cookie.value}`
        },
        body: JSON.stringify({ 
          source: 'chrome_extension', 
          tickets: message.tickets 
        })
      })
      .then(res => {
        if (!res.ok) {
          if (res.status === 401) {
            throw new Error(`Your Gen Con Planner session has expired. Please open ${SERVER_URL} in a new tab to refresh your login, then try again.`);
          }
          return res.json().then(err => { throw new Error(err.error || `HTTP error ${res.status}`); });
        }
        return res.json();
      })
      .then(data => {
        sendResponse({ status: 'success', data });
      })
      .catch(err => {
        sendResponse({ status: 'error', message: err.toString() });
      });
    });

    // Return true to indicate that sendResponse will be called asynchronously
    return true;
  }
});

