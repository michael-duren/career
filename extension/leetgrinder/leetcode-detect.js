// Runs in LeetCode's page world (manifest "world": "MAIN") at document_start.
// All knowledge of LeetCode's undocumented submission endpoint and result DOM
// lives in this file, so a LeetCode change only needs fixing here.
//
// It reports an Accepted submission to the content script with
// window.postMessage, preceded by a "submission" message carrying the judged
// code and language for every submission whose check finishes. Page scripts
// can forge such messages, so the content script validates their shape and
// sizes and treats them only as a hint to open the confirm panel; nothing is
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

  // Code capture: POST /problems/<slug>/submit/ sends {lang, typed_code} and
  // answers {submission_id}. The code is kept per submission id until that
  // submission's check reports a result, then handed to the content script.
  const SUBMIT = /^\/problems\/([a-z0-9]+(?:-[a-z0-9]+)*)\/submit\/?$/;
  const MAX_PENDING = 10;
  const submissions = new Map();
  const delivered = new Set();

  function submitSlug(method, url) {
    if (String(method || "GET").toUpperCase() !== "POST") return null;
    try {
      const m = SUBMIT.exec(new URL(url, window.location.href).pathname);
      return m ? m[1] : null;
    } catch {
      return null;
    }
  }

  // remember records a submission once LeetCode has answered with its id.
  function remember(slug, requestBody, response) {
    let sent;
    try {
      sent = JSON.parse(requestBody);
    } catch {
      return;
    }
    if (!sent || typeof sent.lang !== "string" || typeof sent.typed_code !== "string") return;
    const raw = response && response.submission_id;
    const id = typeof raw === "number" && Number.isSafeInteger(raw) && raw >= 0 ? String(raw) : typeof raw === "string" && /^\d{1,20}$/.test(raw) ? raw : null;
    if (!id) return;
    submissions.set(id, { slug, lang: sent.lang, code: sent.typed_code });
    while (submissions.size > MAX_PENDING) submissions.delete(submissions.keys().next().value);
  }

  function inspect(id, body) {
    if (!body || body.state !== "SUCCESS") return;
    const status = typeof body.status_msg === "string" ? body.status_msg.slice(0, 64) : "";
    const captured = submissions.get(id);
    if (captured && !delivered.has(id)) {
      delivered.add(id);
      submissions.delete(id);
      // Posted before "accepted" so the panel can attach this code.
      window.postMessage({ source: SOURCE, type: "submission", slug: captured.slug, submissionId: id, status, lang: captured.lang, code: captured.code }, window.location.origin);
    }
    if (status === "Accepted") report(`submission-${id}`, "network");
  }

  // Network: wrap fetch and XMLHttpRequest without changing their behavior.
  const originalFetch = window.fetch;
  if (typeof originalFetch === "function") {
    window.fetch = function (...args) {
      const [input, init] = args;
      let submit = null;
      try {
        const isRequest = typeof Request === "function" && input instanceof Request;
        const url = isRequest ? input.url : String(input);
        const slug = submitSlug((init && init.method) || (isRequest ? input.method : "GET"), url);
        if (slug) {
          const body = init && init.body !== undefined ? init.body : null;
          // Read a Request's body from a clone before fetch consumes it.
          const text = typeof body === "string" ? Promise.resolve(body) : isRequest && body === null ? input.clone().text() : null;
          if (text) submit = { slug, text };
        }
      } catch {
        submit = null;
      }
      const promise = originalFetch.apply(this, args);
      promise
        .then((res) => {
          if (submit) {
            Promise.all([submit.text, res.clone().json()]).then(([text, body]) => remember(submit.slug, text, body), () => {});
            return;
          }
          const id = checkId(res.url || (typeof input === "string" ? input : input && input.url) || "");
          if (id) res.clone().json().then((body) => inspect(id, body), () => {});
        })
        .catch(() => {});
      return promise;
    };
  }

  const readJSON = (xhr) => (xhr.responseType === "json" ? xhr.response : JSON.parse(xhr.responseText));

  const originalOpen = XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open = function (method, url, ...rest) {
    const id = checkId(String(url));
    this.__leetgrinderSubmit = submitSlug(method, String(url));
    if (id) {
      this.addEventListener("load", () => {
        try {
          inspect(id, readJSON(this));
        } catch {
          // Not JSON; ignore.
        }
      });
    }
    return originalOpen.call(this, method, url, ...rest);
  };

  const originalSend = XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.send = function (body) {
    const slug = this.__leetgrinderSubmit;
    if (slug && typeof body === "string") {
      this.addEventListener("load", () => {
        try {
          remember(slug, body, readJSON(this));
        } catch {
          // Not JSON; ignore.
        }
      }, { once: true });
    }
    return originalSend.apply(this, arguments);
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
