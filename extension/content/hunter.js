// Hunter - Git Blame for AI
// Content script for GitHub integration

(function() {
  'use strict';

  // Configuration
  const NOTES_REF = 'ai';
  const AI_SESSION_TRAILER = 'AI-Session';
  const AI_SLUG_TRAILER = 'AI-Slug';

  // State
  let cachedNotes = new Map();
  let cachedCommitTrailers = new Map();
  let currentPopup = null;
  let observer = null;

  // Initialize
  function init() {
    console.log('[Hunter] Initializing...');

    // Detect page type and apply appropriate enhancements
    detectAndEnhance();

    // Watch for navigation (GitHub uses SPA-style navigation)
    observeNavigation();
  }

  function detectAndEnhance() {
    const path = window.location.pathname;

    if (path.includes('/pull/') && path.includes('/files')) {
      console.log('[Hunter] Detected PR files page');
      enhancePRFiles();
    } else if (path.includes('/blame/')) {
      console.log('[Hunter] Detected blame page');
      enhanceBlamePage();
    } else if (path.includes('/commit/')) {
      console.log('[Hunter] Detected commit page');
      enhanceCommitPage();
    }
  }

  function observeNavigation() {
    // GitHub uses turbo/pjax for navigation
    document.addEventListener('turbo:load', detectAndEnhance);
    document.addEventListener('pjax:end', detectAndEnhance);

    // Also watch for DOM changes (for dynamic content loading)
    if (observer) {
      observer.disconnect();
    }

    observer = new MutationObserver((mutations) => {
      for (const mutation of mutations) {
        if (mutation.type === 'childList' && mutation.addedNodes.length > 0) {
          // Check if new diff content was added
          for (const node of mutation.addedNodes) {
            if (node.nodeType === Node.ELEMENT_NODE) {
              if (node.classList?.contains('diff-table') ||
                  node.querySelector?.('.diff-table')) {
                detectAndEnhance();
                return;
              }
            }
          }
        }
      }
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true
    });
  }

  // Extract repo info from URL
  function getRepoInfo() {
    const match = window.location.pathname.match(/^\/([^/]+)\/([^/]+)/);
    if (match) {
      return { owner: match[1], repo: match[2] };
    }
    return null;
  }

  // Fetch commit info from GitHub API
  async function fetchCommitInfo(owner, repo, sha) {
    const cacheKey = `${owner}/${repo}/${sha}`;
    if (cachedCommitTrailers.has(cacheKey)) {
      return cachedCommitTrailers.get(cacheKey);
    }

    try {
      const response = await fetch(`https://api.github.com/repos/${owner}/${repo}/commits/${sha}`);
      if (!response.ok) return null;

      const data = await response.json();
      const message = data.commit?.message || '';

      const trailers = parseTrailers(message);
      cachedCommitTrailers.set(cacheKey, trailers);
      return trailers;
    } catch (e) {
      console.error('[Hunter] Failed to fetch commit:', e);
      return null;
    }
  }

  // Fetch git notes from GitHub API
  async function fetchNotes(owner, repo, sha) {
    const cacheKey = `${owner}/${repo}/${sha}`;
    if (cachedNotes.has(cacheKey)) {
      return cachedNotes.get(cacheKey);
    }

    try {
      // Try to fetch the note blob
      // Notes are stored at refs/notes/ai with the note content keyed by commit SHA
      const response = await fetch(
        `https://api.github.com/repos/${owner}/${repo}/git/refs/notes/${NOTES_REF}`
      );

      if (!response.ok) {
        cachedNotes.set(cacheKey, null);
        return null;
      }

      const refData = await response.json();
      const notesTreeSha = refData.object?.sha;

      if (!notesTreeSha) {
        cachedNotes.set(cacheKey, null);
        return null;
      }

      // Get the tree to find the note for this commit
      const treeResponse = await fetch(
        `https://api.github.com/repos/${owner}/${repo}/git/trees/${notesTreeSha}`
      );

      if (!treeResponse.ok) {
        cachedNotes.set(cacheKey, null);
        return null;
      }

      const treeData = await treeResponse.json();
      const noteEntry = treeData.tree?.find(entry => entry.path === sha);

      if (!noteEntry) {
        cachedNotes.set(cacheKey, null);
        return null;
      }

      // Fetch the note content
      const blobResponse = await fetch(
        `https://api.github.com/repos/${owner}/${repo}/git/blobs/${noteEntry.sha}`
      );

      if (!blobResponse.ok) {
        cachedNotes.set(cacheKey, null);
        return null;
      }

      const blobData = await blobResponse.json();
      const content = atob(blobData.content);
      const noteData = JSON.parse(content);

      cachedNotes.set(cacheKey, noteData);
      return noteData;
    } catch (e) {
      console.error('[Hunter] Failed to fetch notes:', e);
      cachedNotes.set(cacheKey, null);
      return null;
    }
  }

  // Parse git trailers from commit message
  function parseTrailers(message) {
    const trailers = {};
    const lines = message.split('\n');

    for (const line of lines) {
      if (line.startsWith(AI_SESSION_TRAILER + ':')) {
        trailers.sessionId = line.substring(AI_SESSION_TRAILER.length + 1).trim();
      } else if (line.startsWith(AI_SLUG_TRAILER + ':')) {
        trailers.slug = line.substring(AI_SLUG_TRAILER.length + 1).trim();
      }
    }

    return trailers.sessionId ? trailers : null;
  }

  // Enhance PR files page
  async function enhancePRFiles() {
    const repoInfo = getRepoInfo();
    if (!repoInfo) return;

    // Find all diff file headers
    const fileHeaders = document.querySelectorAll('.file-header');

    for (const header of fileHeaders) {
      // Skip if already processed
      if (header.querySelector('.hunter-ai-badge')) continue;

      // Get the diff table for this file
      const diffContainer = header.closest('.file');
      if (!diffContainer) continue;

      // Find all line rows in this diff
      const lineRows = diffContainer.querySelectorAll('tr.blob-code-addition, tr.blob-code-deletion, tr[data-hunk]');

      // Collect unique commits from blame info or commit links
      const commitShas = new Set();
      const blameLinks = diffContainer.querySelectorAll('a[href*="/commit/"]');

      for (const link of blameLinks) {
        const match = link.href.match(/\/commit\/([a-f0-9]+)/);
        if (match) {
          commitShas.add(match[1]);
        }
      }

      // Check each commit for AI trailers
      let aiLineCount = 0;
      let totalLines = lineRows.length;
      const aiCommits = new Set();

      for (const sha of commitShas) {
        const trailers = await fetchCommitInfo(repoInfo.owner, repoInfo.repo, sha);
        if (trailers?.sessionId) {
          aiCommits.add(sha);
        }
      }

      // If we found AI commits, add badge to header
      if (aiCommits.size > 0) {
        addAIBadge(header, aiCommits.size, commitShas.size);
      }
    }
  }

  // Enhance blame page
  async function enhanceBlamePage() {
    const repoInfo = getRepoInfo();
    if (!repoInfo) return;

    // Find all blame hunk headers (contain commit info)
    const blameHunks = document.querySelectorAll('.blame-hunk');

    for (const hunk of blameHunks) {
      // Skip if already processed
      if (hunk.querySelector('.hunter-ai-indicator')) continue;

      // Find the commit SHA link
      const commitLink = hunk.querySelector('a[href*="/commit/"]');
      if (!commitLink) continue;

      const match = commitLink.href.match(/\/commit\/([a-f0-9]+)/);
      if (!match) continue;

      const sha = match[1];

      // Check for AI trailers
      const trailers = await fetchCommitInfo(repoInfo.owner, repoInfo.repo, sha);
      if (!trailers?.sessionId) continue;

      // Add AI indicator
      const indicator = createAIIndicator(trailers, repoInfo, sha);
      const messageCell = hunk.querySelector('.blame-commit-message') || hunk.querySelector('.blob-code-hunk');

      if (messageCell) {
        messageCell.insertBefore(indicator, messageCell.firstChild);
      }

      // Highlight the lines in this hunk
      const codeLines = hunk.querySelectorAll('.blob-code-inner');
      for (const line of codeLines) {
        line.classList.add('hunter-ai-line');
      }
    }
  }

  // Enhance commit page
  async function enhanceCommitPage() {
    const repoInfo = getRepoInfo();
    if (!repoInfo) return;

    // Get commit SHA from URL
    const match = window.location.pathname.match(/\/commit\/([a-f0-9]+)/);
    if (!match) return;

    const sha = match[1];

    // Check for AI trailers
    const trailers = await fetchCommitInfo(repoInfo.owner, repoInfo.repo, sha);
    if (!trailers?.sessionId) return;

    // Add AI banner to commit header
    const commitHeader = document.querySelector('.commit-title') || document.querySelector('.commit-desc');
    if (commitHeader && !commitHeader.querySelector('.hunter-ai-badge')) {
      const badge = document.createElement('span');
      badge.className = 'hunter-ai-badge';
      badge.innerHTML = '<span class="hunter-ai-badge-icon">🤖</span>AI-Assisted';
      badge.style.cursor = 'pointer';
      badge.addEventListener('click', () => showPopup(badge, trailers, repoInfo, sha));
      commitHeader.appendChild(badge);
    }
  }

  // Create AI indicator element
  function createAIIndicator(trailers, repoInfo, sha) {
    const indicator = document.createElement('span');
    indicator.className = 'hunter-ai-indicator';
    indicator.title = `AI Session: ${trailers.sessionId.substring(0, 8)}`;
    indicator.addEventListener('click', (e) => {
      e.preventDefault();
      e.stopPropagation();
      showPopup(indicator, trailers, repoInfo, sha);
    });
    return indicator;
  }

  // Add AI badge to file header
  function addAIBadge(header, aiCount, totalCount) {
    const badge = document.createElement('span');
    badge.className = 'hunter-ai-badge';
    badge.innerHTML = `<span class="hunter-ai-badge-icon">🤖</span>AI: ${aiCount}/${totalCount} commits`;

    const titleContainer = header.querySelector('.file-info') || header;
    titleContainer.appendChild(badge);
  }

  // Show popup with session details
  async function showPopup(anchor, trailers, repoInfo, sha) {
    // Close existing popup
    closePopup();

    // Try to fetch notes for more details
    const notes = await fetchNotes(repoInfo.owner, repoInfo.repo, sha);

    // Create popup
    const popup = document.createElement('div');
    popup.className = 'hunter-popup';

    const shortId = trailers.sessionId.substring(0, 8);

    popup.innerHTML = `
      <div class="hunter-popup-header">
        <span class="hunter-popup-header-icon">🤖</span>
        <span class="hunter-popup-header-title">AI Session</span>
        <span class="hunter-popup-header-session">${shortId}</span>
      </div>
      <div class="hunter-popup-content">
        ${trailers.slug ? `
          <div class="hunter-popup-section">
            <div class="hunter-popup-section-title">Session</div>
            <div>${trailers.slug}</div>
          </div>
        ` : ''}
        ${notes?.plan ? `
          <div class="hunter-popup-section">
            <div class="hunter-popup-section-title">Plan</div>
            <div class="hunter-popup-plan">${escapeHtml(truncate(notes.plan, 500))}</div>
          </div>
        ` : ''}
        ${notes?.summary ? `
          <div class="hunter-popup-section">
            <div class="hunter-popup-section-title">Summary</div>
            <div class="hunter-popup-summary">${escapeHtml(notes.summary)}</div>
          </div>
        ` : ''}
        ${!notes?.plan && !notes?.summary ? `
          <div class="hunter-popup-section">
            <div class="hunter-popup-summary" style="color: #57606a;">
              No additional details available.<br>
              Run <code>hunter enrich</code> locally to add plan and summary.
            </div>
          </div>
        ` : ''}
      </div>
      <div class="hunter-popup-footer">
        <button class="hunter-popup-btn hunter-copy-btn">
          <span class="hunter-popup-btn-icon">📋</span>
          Copy ID
        </button>
        <button class="hunter-popup-btn hunter-restore-btn">
          <span class="hunter-popup-btn-icon">📂</span>
          Open Locally
        </button>
      </div>
    `;

    // Position popup
    const rect = anchor.getBoundingClientRect();
    popup.style.position = 'fixed';
    popup.style.top = `${rect.bottom + 8}px`;
    popup.style.left = `${Math.max(8, rect.left - 100)}px`;

    // Ensure popup doesn't go off-screen
    document.body.appendChild(popup);
    const popupRect = popup.getBoundingClientRect();
    if (popupRect.right > window.innerWidth - 8) {
      popup.style.left = `${window.innerWidth - popupRect.width - 8}px`;
    }
    if (popupRect.bottom > window.innerHeight - 8) {
      popup.style.top = `${rect.top - popupRect.height - 8}px`;
    }

    // Add event listeners
    popup.querySelector('.hunter-copy-btn').addEventListener('click', () => {
      navigator.clipboard.writeText(trailers.sessionId);
      popup.querySelector('.hunter-copy-btn').innerHTML = '<span class="hunter-popup-btn-icon">✓</span>Copied!';
    });

    popup.querySelector('.hunter-restore-btn').addEventListener('click', () => {
      const command = `hunter restore ${shortId}`;
      navigator.clipboard.writeText(command);
      popup.querySelector('.hunter-restore-btn').innerHTML = '<span class="hunter-popup-btn-icon">✓</span>Command Copied!';
    });

    // Close on click outside
    currentPopup = popup;
    setTimeout(() => {
      document.addEventListener('click', handleOutsideClick);
    }, 0);
  }

  function closePopup() {
    if (currentPopup) {
      currentPopup.remove();
      currentPopup = null;
      document.removeEventListener('click', handleOutsideClick);
    }
  }

  function handleOutsideClick(e) {
    if (currentPopup && !currentPopup.contains(e.target)) {
      closePopup();
    }
  }

  // Utility functions
  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }

  function truncate(text, maxLength) {
    if (text.length <= maxLength) return text;
    return text.substring(0, maxLength) + '...';
  }

  // Initialize when DOM is ready
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
