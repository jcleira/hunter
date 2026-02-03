// Hunter - Popup Script

document.addEventListener('DOMContentLoaded', () => {
  // Load settings
  chrome.storage.sync.get(['enabled', 'showIndicators', 'showBadges'], (result) => {
    document.getElementById('enabled').checked = result.enabled !== false;
    document.getElementById('showIndicators').checked = result.showIndicators !== false;
    document.getElementById('showBadges').checked = result.showBadges !== false;
  });

  // Save settings on change
  document.getElementById('enabled').addEventListener('change', (e) => {
    chrome.storage.sync.set({ enabled: e.target.checked });
    notifyContentScript();
  });

  document.getElementById('showIndicators').addEventListener('change', (e) => {
    chrome.storage.sync.set({ showIndicators: e.target.checked });
    notifyContentScript();
  });

  document.getElementById('showBadges').addEventListener('change', (e) => {
    chrome.storage.sync.set({ showBadges: e.target.checked });
    notifyContentScript();
  });

  // Check current page status
  chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
    const tab = tabs[0];
    const url = tab?.url || '';

    const statusDot = document.querySelector('.status-dot');
    const statusText = document.getElementById('status-text');

    if (url.includes('github.com')) {
      statusDot.classList.remove('inactive');
      statusDot.classList.add('active');

      if (url.includes('/pull/') && url.includes('/files')) {
        statusText.textContent = 'PR Files - Hunter active';
      } else if (url.includes('/blame/')) {
        statusText.textContent = 'Blame view - Hunter active';
      } else if (url.includes('/commit/')) {
        statusText.textContent = 'Commit view - Hunter active';
      } else {
        statusText.textContent = 'GitHub detected';
      }
    } else {
      statusDot.classList.remove('active');
      statusDot.classList.add('inactive');
      statusText.textContent = 'Not a GitHub page';
    }
  });
});

function notifyContentScript() {
  chrome.tabs.query({ active: true, currentWindow: true }, (tabs) => {
    if (tabs[0]?.id) {
      chrome.tabs.sendMessage(tabs[0].id, { type: 'SETTINGS_CHANGED' });
    }
  });
}
