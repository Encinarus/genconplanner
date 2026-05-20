/**
 * Gen Con Planner Extension - Content Script for gencon.com
 * 
 * Modifies event detail pages (e.g., https://www.gencon.com/events/<number>)
 * by inserting an <hr> after the last "Myself" or "Another ticket for me" option,
 * and sorting the remaining ticket options alphabetically by their label text.
 */

function modifyTicketPurchase() {
  const containerDiv = document.getElementById('event_detail_ticket_purchase');
  if (!containerDiv) return false;

  // Avoid duplicate modifications if observer triggers multiple times
  if (containerDiv.dataset.modifiedByGenconPlanner) return true;

  const form = containerDiv.querySelector('#user_tickets form');
  if (!form) return false;

  // Ensure the elements following the ticket list (e.g., table, action buttons, or divider)
  // have been parsed and attached to the DOM. This guarantees all ticket spans are fully loaded
  // before we attempt to sort them.
  const postListElement = form.querySelector('table, div, a, hr');
  if (!postListElement) return false;

  const formP = form.querySelector('p');
  if (!formP) return false;

  const spans = Array.from(formP.querySelectorAll('span'));
  if (spans.length === 0) return false;

  // Mark as modified
  containerDiv.dataset.modifiedByGenconPlanner = 'true';

  // Group each span with its trailing text nodes (whitespace) and <br> sibling
  const spanData = spans.map(span => {
    let next = span.nextSibling;
    const siblings = [];
    while (next && (next.nodeType === Node.TEXT_NODE || (next.nodeType === Node.ELEMENT_NODE && next.tagName.toLowerCase() === 'br'))) {
      siblings.push(next);
      if (next.nodeType === Node.ELEMENT_NODE && next.tagName.toLowerCase() === 'br') {
        break;
      }
      next = next.nextSibling;
    }

    // Extract person ID from checkbox name (tickets_for[120344]) or label for (tickets_for_120344)
    const checkbox = span.querySelector('input[type="checkbox"]');
    const label = span.querySelector('label');
    let personId = null;

    if (checkbox) {
      const name = checkbox.getAttribute('name') || '';
      const nameMatch = name.match(/^tickets_for\[(\d+)\]$/);
      if (nameMatch) personId = nameMatch[1];
    }
    if (!personId && label) {
      const htmlFor = label.getAttribute('for') || '';
      const forMatch = htmlFor.match(/^tickets_for_(\d+)$/);
      if (forMatch) personId = forMatch[1];
    }

    // Append the ID to the label text
    if (personId && label) {
      const currentText = label.textContent.trim();
      if (!currentText.endsWith(`(${personId})`)) {
        label.textContent = `${currentText} (${personId})`;
      }
    }

    return { span, siblings };
  });

  // Find the last span representing "Myself" or "Another ticket for me"
  let lastSelfSpanIndex = -1;
  for (let i = 0; i < spanData.length; i++) {
    const text = spanData[i].span.textContent.trim().toLowerCase();
    if (text === 'myself' || text.includes('another ticket for me')) {
      lastSelfSpanIndex = i;
    }
  }

  // Helper function to extract the label text for sorting
  function getLabelText(span) {
    const label = span.querySelector('label');
    if (label) {
      return label.textContent.trim();
    }
    return span.textContent.trim();
  }

  // Separate the remaining spans and sort them alphabetically by label text
  const remainingSpans = spanData.slice(lastSelfSpanIndex + 1);
  remainingSpans.sort((a, b) => {
    const textA = getLabelText(a.span).toLowerCase();
    const textB = getLabelText(b.span).toLowerCase();
    return textA.localeCompare(textB);
  });

  // Reinsert elements into the DOM
  if (lastSelfSpanIndex !== -1) {
    const lastSelf = spanData[lastSelfSpanIndex];
    let insertAfterTarget = lastSelf.siblings.length > 0 ? lastSelf.siblings[lastSelf.siblings.length - 1] : lastSelf.span;

    if (remainingSpans.length > 0) {
      // Insert <hr> after the last self ticket option
      const hr = document.createElement('hr');
      hr.setAttribute('style', 'margin:0 5px 0 5px;');
      insertAfterTarget.after(hr);
      insertAfterTarget = hr;

      // Insert sorted remaining spans and their <br> siblings
      for (const item of remainingSpans) {
        insertAfterTarget.after(item.span);
        insertAfterTarget = item.span;
        for (const sib of item.siblings) {
          insertAfterTarget.after(sib);
          insertAfterTarget = sib;
        }
      }
    }
  } else {
    // If no self spans exist, just append the sorted remaining spans to the container `<p>`
    for (const item of remainingSpans) {
      formP.appendChild(item.span);
      for (const sib of item.siblings) {
        formP.appendChild(sib);
      }
    }
  }

  return true;
}

/**
 * Parses the Gen Con transactions page for tickets purchased this year.
 * Identifies transaction blocks, filters by current year, extracts event IDs,
 * ticket IDs, purchaser, recipient, and ticket type.
 */
function parseTransactionsPage() {
  if (!window.location.href.includes('/transactions') && !window.location.href.includes('/my_transactions')) return false;

  // Avoid duplicate injection
  if (document.getElementById('genconplanner-sync-banner')) return true;

  const currentYear = new Date().getFullYear().toString();
  const tickets = [];

  // Purchaser Name: Extract from page title e.g. "Alek Dembowski's Transactions"
  let purchaserName = "Myself";
  const pageTitleEl = document.querySelector('.page-title, h1');
  if (pageTitleEl && pageTitleEl.textContent.trim()) {
    const titleText = pageTitleEl.textContent.trim();
    const match = titleText.match(/^(.+?)(?:&#39;|')s\s+Transactions/i);
    if (match && match[1].trim()) {
      purchaserName = match[1].trim();
    } else {
      const parts = titleText.split(':');
      purchaserName = parts[parts.length - 1].trim();
    }
  }

  // Find all transaction panels
  const panels = Array.from(document.querySelectorAll('.panel'));

  for (const panel of panels) {
    const titlebar = panel.querySelector('.panel_titlebar');
    if (!titlebar) continue;

    const titlebarText = titlebar.textContent.trim();
    // e.g. "Transaction: 2026/05/17 10:56 PM"
    const yearMatch = titlebarText.match(/\b(202\d)\b/);
    if (!yearMatch || yearMatch[1] !== currentYear) continue;

    // Extract transaction ID from panel ID e.g. "txn-3684243"
    let currentTransactionId = panel.id ? panel.id.replace('txn-', '') : "";
    if (!currentTransactionId) {
      const txMatch = titlebarText.match(/(?:Transaction|TXN)[:#\s]*(\d+)/i);
      if (txMatch) currentTransactionId = txMatch[1];
    }

    // Find all rows in this panel's records table
    const rows = Array.from(panel.querySelectorAll('table.records tr'));
    const panelTickets = [];

    for (const row of rows) {
      const cells = Array.from(row.querySelectorAll('td'));
      if (cells.length < 2) continue; // Skip header row or malformed rows

      const descCellText = cells[0].textContent.trim();
      const recipientCellText = cells[1].textContent.trim();

      const isPurchase = descCellText.includes('Ticket Purchase');
      const isReturn = descCellText.includes('Ticket Return') || descCellText.includes('Ticket Cancellation');
      if (!isPurchase && !isReturn) continue;

      // Extract Event ID e.g. "- BGM26ND319431 (Gempire on Wednesday..."
      const eventIdMatch = descCellText.match(/\b([A-Z]{3}\d{2}[A-Z]{2}\d+)\b/);
      if (!eventIdMatch) continue;

      const eventId = eventIdMatch[1];

      let recipientName = recipientCellText || purchaserName;
      if (recipientName.includes('Another ticket for me')) {
        recipientName = 'Another ticket for me';
      }

      let ticketType = 'physical';
      if (descCellText.toLowerCase().includes('e-ticket') || descCellText.toLowerCase().includes('eticket') || descCellText.toLowerCase().includes('electronic')) {
        ticketType = 'eticket';
      }

      panelTickets.push({
        eventId,
        purchaserGenconId: "",
        purchaserName,
        recipientGenconId: "",
        recipientName,
        ticketType,
        status: isReturn ? 'returned' : 'active'
      });
    }

    // Deterministically sort panel tickets by Event ID, then Recipient Name, then Ticket Type, then Status
    // This guarantees that even if Gen Con renders table rows in an arbitrary or shifting order,
    // the generated genconTicketId (e.g. TXN-3684243-1, TXN-3684243-2) will remain 100% stable and locked
    // to the exact same recipient across repeated imports!
    panelTickets.sort((a, b) => {
      if (a.eventId !== b.eventId) return a.eventId.localeCompare(b.eventId);
      if (a.recipientName !== b.recipientName) return a.recipientName.localeCompare(b.recipientName);
      if (a.ticketType !== b.ticketType) return a.ticketType.localeCompare(b.ticketType);
      return a.status.localeCompare(b.status);
    });

    let ticketIndex = 1;
    for (const pt of panelTickets) {
      let genconTicketId = currentTransactionId ? `TXN-${currentTransactionId}-${ticketIndex}` : `TXN-${pt.eventId}-${tickets.length + 1}`;
      ticketIndex++;

      if (!tickets.some(t => t.eventId === pt.eventId && t.genconTicketId === genconTicketId && t.recipientName === pt.recipientName)) {
        tickets.push({
          eventId: pt.eventId,
          genconTicketId,
          purchaserGenconId: pt.purchaserGenconId,
          purchaserName: pt.purchaserName,
          recipientGenconId: pt.recipientGenconId,
          recipientName: pt.recipientName,
          ticketType: pt.ticketType
        });
      }
    }
  }

  createSyncBanner(tickets, currentYear);
  return true;
}

function createSyncBanner(tickets, year) {
  const pageTitleEl = document.querySelector('.page-title, h1');
  if (!pageTitleEl) return;

  // Avoid duplicate injection (recheck here just in case)
  if (document.getElementById('genconplanner-sync-banner')) return;

  chrome.storage.local.get({ syncTargetEnv: 'prod' }, (result) => {
    let targetEnv = result.syncTargetEnv;

    const container = document.createElement('div');
    container.id = 'genconplanner-sync-banner';
    container.style.cssText = `
      display: flex;
      align-items: center;
      gap: 12px;
      font-family: 'Inter', system-ui, -apple-system, sans-serif;
      font-size: 14px;
      font-weight: normal;
    `;

    const statusText = document.createElement('div');
    statusText.style.cssText = 'font-size: 14px; font-weight: 500;';

    if (tickets.length === 0) {
      statusText.style.color = '#64748b';
      statusText.textContent = `No ${year} tickets found`;
      container.appendChild(statusText);
    } else {
      statusText.style.color = '#64748b';
      statusText.textContent = `${tickets.length} ticket(s) found`;

      // Create environment toggle container
      const toggleContainer = document.createElement('div');
      toggleContainer.style.cssText = `
        display: inline-flex;
        background: #f1f5f9;
        border: 1px solid #cbd5e1;
        border-radius: 8px;
        padding: 2px;
        gap: 2px;
        user-select: none;
      `;

      const localBtn = document.createElement('button');
      localBtn.textContent = 'Local Dev';
      const prodBtn = document.createElement('button');
      prodBtn.textContent = 'Production';

      const btnStyleBase = `
        border: none;
        padding: 4px 8px;
        border-radius: 6px;
        font-size: 11px;
        font-weight: 600;
        cursor: pointer;
        transition: all 0.15s ease;
        font-family: inherit;
      `;

      const setActiveStyle = (btn) => {
        btn.style.cssText = btnStyleBase + `
          background: #0284c7;
          color: white;
          box-shadow: 0 1px 2px rgba(0,0,0,0.05);
        `;
      };

      const setInactiveStyle = (btn) => {
        btn.style.cssText = btnStyleBase + `
          background: transparent;
          color: #64748b;
        `;
      };

      const updateToggleUI = () => {
        if (targetEnv === 'local') {
          setActiveStyle(localBtn);
          setInactiveStyle(prodBtn);
        } else {
          setActiveStyle(prodBtn);
          setInactiveStyle(localBtn);
        }
      };

      localBtn.onclick = () => {
        targetEnv = 'local';
        chrome.storage.local.set({ syncTargetEnv: 'local' });
        updateToggleUI();
      };

      prodBtn.onclick = () => {
        targetEnv = 'prod';
        chrome.storage.local.set({ syncTargetEnv: 'prod' });
        updateToggleUI();
      };

      updateToggleUI();
      toggleContainer.appendChild(localBtn);
      toggleContainer.appendChild(prodBtn);

      const syncBtn = document.createElement('button');
      syncBtn.style.cssText = `
        background: #0284c7;
        color: white;
        border: none;
        padding: 8px 16px;
        border-radius: 8px;
        font-weight: 600;
        font-size: 14px;
        cursor: pointer;
        transition: all 0.2s ease;
        box-shadow: 0 4px 12px rgba(2, 132, 199, 0.3);
        display: flex;
        align-items: center;
        gap: 8px;
      `;
      syncBtn.textContent = 'Import current year to genconplanner';
      syncBtn.onmouseover = () => { if (!syncBtn.disabled) syncBtn.style.background = '#0369a1'; };
      syncBtn.onmouseout = () => { if (!syncBtn.disabled) syncBtn.style.background = '#0284c7'; };

      syncBtn.onclick = () => {
        syncBtn.disabled = true;
        syncBtn.style.background = '#64748b';
        syncBtn.textContent = 'Importing...';
        statusText.textContent = '';

        // Disable toggles while sync is active
        localBtn.disabled = true;
        prodBtn.disabled = true;

        chrome.runtime.sendMessage({ action: 'sync_tickets', tickets, targetEnv }, (response) => {
          localBtn.disabled = false;
          prodBtn.disabled = false;

          if (response && response.status === 'success') {
            syncBtn.style.background = '#16a34a';
            syncBtn.textContent = 'Imported Successfully!';
            statusText.style.color = '#16a34a';
            statusText.textContent = `Imported ${response.data.syncedCount || tickets.length} tickets`;
          } else {
            syncBtn.disabled = false;
            syncBtn.style.background = '#0284c7';
            syncBtn.textContent = 'Try Import Again';
            statusText.style.color = '#dc2626';
            statusText.textContent = response ? response.message : 'Failed to connect to background worker.';
          }
        });
      };

      container.appendChild(statusText);
      container.appendChild(toggleContainer);
      container.appendChild(syncBtn);
    }

    pageTitleEl.style.display = 'flex';
    pageTitleEl.style.justifyContent = 'space-between';
    pageTitleEl.style.alignItems = 'center';
    pageTitleEl.appendChild(container);
  });
}

// Attempt immediate execution based on URL
if (window.location.href.includes('/transactions') || window.location.href.includes('/my_transactions')) {
  if (!parseTransactionsPage()) {
    const observer = new MutationObserver((mutations, obs) => {
      if (parseTransactionsPage()) {
        obs.disconnect();
      }
    });
    observer.observe(document.documentElement, { childList: true, subtree: true });
    setTimeout(() => observer.disconnect(), 10000);
  }
} else if (window.location.href.includes('/events/')) {
  if (!modifyTicketPurchase()) {
    const observer = new MutationObserver((mutations, obs) => {
      if (modifyTicketPurchase()) {
        obs.disconnect();
      }
    });
    observer.observe(document.documentElement, { childList: true, subtree: true });
    setTimeout(() => observer.disconnect(), 10000);
  }
}

