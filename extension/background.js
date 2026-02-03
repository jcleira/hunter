// Hunter - Background Service Worker
// Handles API caching and cross-tab coordination

const CACHE_DURATION = 5 * 60 * 1000; // 5 minutes
const cache = new Map();

// Clean up expired cache entries periodically
setInterval(() => {
  const now = Date.now();
  for (const [key, value] of cache.entries()) {
    if (now - value.timestamp > CACHE_DURATION) {
      cache.delete(key);
    }
  }
}, 60 * 1000);

// Handle messages from content scripts
chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  if (message.type === 'FETCH_COMMIT') {
    handleFetchCommit(message, sendResponse);
    return true; // Async response
  }

  if (message.type === 'FETCH_NOTES') {
    handleFetchNotes(message, sendResponse);
    return true;
  }

  if (message.type === 'GET_SETTINGS') {
    chrome.storage.sync.get(['enabled', 'showIndicators', 'showBadges'], (result) => {
      sendResponse({
        enabled: result.enabled !== false,
        showIndicators: result.showIndicators !== false,
        showBadges: result.showBadges !== false
      });
    });
    return true;
  }
});

async function handleFetchCommit(message, sendResponse) {
  const { owner, repo, sha } = message;
  const cacheKey = `commit:${owner}/${repo}/${sha}`;

  // Check cache
  const cached = cache.get(cacheKey);
  if (cached && Date.now() - cached.timestamp < CACHE_DURATION) {
    sendResponse({ data: cached.data });
    return;
  }

  try {
    const response = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/commits/${sha}`,
      {
        headers: {
          'Accept': 'application/vnd.github.v3+json'
        }
      }
    );

    if (!response.ok) {
      sendResponse({ error: `HTTP ${response.status}` });
      return;
    }

    const data = await response.json();
    cache.set(cacheKey, { data, timestamp: Date.now() });
    sendResponse({ data });
  } catch (e) {
    sendResponse({ error: e.message });
  }
}

async function handleFetchNotes(message, sendResponse) {
  const { owner, repo, sha } = message;
  const cacheKey = `notes:${owner}/${repo}/${sha}`;

  // Check cache
  const cached = cache.get(cacheKey);
  if (cached && Date.now() - cached.timestamp < CACHE_DURATION) {
    sendResponse({ data: cached.data });
    return;
  }

  try {
    // First, get the notes ref
    const refResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/refs/notes/ai`,
      {
        headers: {
          'Accept': 'application/vnd.github.v3+json'
        }
      }
    );

    if (!refResponse.ok) {
      cache.set(cacheKey, { data: null, timestamp: Date.now() });
      sendResponse({ data: null });
      return;
    }

    const refData = await refResponse.json();
    const notesTreeSha = refData.object?.sha;

    if (!notesTreeSha) {
      cache.set(cacheKey, { data: null, timestamp: Date.now() });
      sendResponse({ data: null });
      return;
    }

    // Get the tree
    const treeResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/trees/${notesTreeSha}`,
      {
        headers: {
          'Accept': 'application/vnd.github.v3+json'
        }
      }
    );

    if (!treeResponse.ok) {
      cache.set(cacheKey, { data: null, timestamp: Date.now() });
      sendResponse({ data: null });
      return;
    }

    const treeData = await treeResponse.json();
    const noteEntry = treeData.tree?.find(entry => entry.path === sha);

    if (!noteEntry) {
      cache.set(cacheKey, { data: null, timestamp: Date.now() });
      sendResponse({ data: null });
      return;
    }

    // Fetch the note blob
    const blobResponse = await fetch(
      `https://api.github.com/repos/${owner}/${repo}/git/blobs/${noteEntry.sha}`,
      {
        headers: {
          'Accept': 'application/vnd.github.v3+json'
        }
      }
    );

    if (!blobResponse.ok) {
      cache.set(cacheKey, { data: null, timestamp: Date.now() });
      sendResponse({ data: null });
      return;
    }

    const blobData = await blobResponse.json();
    const content = atob(blobData.content);
    const noteData = JSON.parse(content);

    cache.set(cacheKey, { data: noteData, timestamp: Date.now() });
    sendResponse({ data: noteData });
  } catch (e) {
    console.error('[Hunter] Background fetch error:', e);
    sendResponse({ error: e.message });
  }
}

// Log installation
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === 'install') {
    console.log('[Hunter] Extension installed');
    // Set default settings
    chrome.storage.sync.set({
      enabled: true,
      showIndicators: true,
      showBadges: true
    });
  }
});
