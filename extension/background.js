/**
 * Gen Con Planner Extension - Background Service Worker
 * 
 * Handles cross-origin requests to the Gen Con Planner server API
 * by dynamically retrieving the user's signinToken cookie (either from local dev 
 * or production) and making the POST request.
 */

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.action === 'sync_tickets') {
    const targetEnv = message.targetEnv || 'prod';
    const serverUrl = targetEnv === 'local' ? 'http://localhost:8080' : 'https://www.genconplanner.com';

    chrome.cookies.get({ url: serverUrl, name: 'signinToken' }, (cookie) => {
      if (!cookie) {
        sendResponse({ 
          status: 'error', 
          message: `Not logged into Gen Con Planner. Please log in at ${serverUrl} first.` 
        });
        return;
      }

      const currentYear = new Date().getFullYear();
      const endpoint = `${serverUrl}/api/v1/party/${currentYear}/tickets/sync`;

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
            throw new Error(`Your Gen Con Planner session has expired. Please open ${serverUrl} in a new tab to refresh your login, then try again.`);
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



