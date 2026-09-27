// Runs in LeetCode's page world (manifest "world": "MAIN") at document_start.
// All knowledge of LeetCode's undocumented submission endpoint and result DOM
// lives in this file, so a LeetCode change only needs fixing here.
//
// It reports an Accepted submission to the content script with
// window.postMessage. Page scripts can forge such messages, so the content
// script treats them only as a hint to open the confirm panel; nothing is
// sent to the app without the learner pressing Submit.
(() => {
  "use strict";
  if (window.__leetgrinderDetect) return;
  window.__leetgrinderDetect = true;

  const SOURCE = "leetgrinder-detect";
  // Submit checks poll /submissions/detail/<numeric id>/check/. "Run code"
  // uses runcode_<...> ids, which this deliberately does not match.
  const CHECK = /\/submissions\/detail\/(\d+)\/check\/?$/;
  const RESULT_SELECTOR = '[data-e2e-locator="submission-result"]';
  const QUIET_MS = 10000;

  let lastReport = 0;
  // Set when the learner submits; enables the DOM fallback for a while.
  let armedUntil = 0;
  const seen = new Set();

  function report(id, via) {
    const now = Date.now();
    if (seen.has(id) || now - lastReport < QUIET_MS) return;
    seen.add(id);
    lastReport = now;
    armedUntil = 0;
    window.postMessage({ source: SOURCE, type: "accepted", submissionId: id, via }, window.location.origin);
  }

  function checkId(url) {
    try {
      const m = CHECK.exec(new URL(url, window.location.href).pathname);
      return m ? m[1] : null;
    } catch {
      return null;
    }
  }

  function inspect(id, body) {
    if (body && body.state === "SUCCESS" && body.status_msg === "Accepted") report(`submission-${id}`, "network");
  }

  // Network: wrap fetch and XMLHttpRequest without changing their behavior.
  const originalFetch = window.fetch;
  if (typeof originalFetch === "function") {
    window.fetch = function (...args) {
      const promise = originalFetch.apply(this, args);
      promise
        .then((res) => {
          const input = args[0];
          const id = checkId(res.url || (typeof input === "string" ? input : input && input.url) || "");
          if (id) res.clone().json().then((body) => inspect(id, body), () => {});
        })
        .catch(() => {});
      return promise;
    };
  }

  const originalOpen = XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open = function (method, url, ...rest) {
    const id = checkId(String(url));
    if (id) {
      this.addEventListener("load", () => {
        try {
          const body = this.responseType === "json" ? this.response : JSON.parse(this.responseText);
          inspect(id, body);
        } catch {
          // Not JSON; ignore.
        }
      });
    }
    return originalOpen.call(this, method, url, ...rest);
  };

  // DOM fallback: the result panel shows "Accepted" after a passing submit.
  // Old submissions render the same element, so it only counts shortly after
  // the learner submits (Submit button or Ctrl/Cmd+Enter). Each result element
  // is reported at most once.
  const SUBMIT_SELECTOR = '[data-e2e-locator="console-submit-button"]';
  const ARM_MS = 5 * 60000;
  document.addEventListener(
    "click",
    (event) => {
      if (event.target instanceof Element && event.target.closest(SUBMIT_SELECTOR)) armedUntil = Date.now() + ARM_MS;
    },
    true,
  );
  document.addEventListener(
    "keydown",
    (event) => {
      if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) armedUntil = Date.now() + ARM_MS;
    },
    true,
  );
  const reported = new WeakSet();
  function scan() {
    for (const el of document.querySelectorAll(RESULT_SELECTOR)) {
      // Results can render as pending first and change text in place.
      if (reported.has(el) || el.textContent.trim() !== "Accepted") continue;
      reported.add(el);
      if (Date.now() < armedUntil) report(`dom-${Date.now()}`, "dom");
    }
  }
  let pending = false;
  new MutationObserver(() => {
    if (pending) return;
    pending = true;
    setTimeout(() => {
      pending = false;
      scan();
    }, 250);
  }).observe(document, { childList: true, subtree: true, characterData: true });
})();
