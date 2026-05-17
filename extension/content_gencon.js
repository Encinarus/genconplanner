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

// Attempt immediate execution
if (!modifyTicketPurchase()) {
  // If the target container is not yet in the DOM, set up a MutationObserver
  const observer = new MutationObserver((mutations, obs) => {
    if (modifyTicketPurchase()) {
      obs.disconnect();
    }
  });

  observer.observe(document.documentElement, { childList: true, subtree: true });

  // Disconnect observer after 10 seconds to prevent memory leaks on pages without ticket purchase
  setTimeout(() => observer.disconnect(), 10000);
}
