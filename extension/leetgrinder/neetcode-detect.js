// Runs in NeetCode's page world (manifest "world": "MAIN") at document_start.
// All knowledge of NeetCode's undocumented submit endpoint lives in this
// file, so a NeetCode change only needs fixing here.
//
// NeetCode's Submit posts {data: {problemId, rawCode, lang}} to
// /api/executeCodeFunctionHttp and answers {data: {status: {description}}}
// once the code is judged ("Run" uses runCodeFunctionHttp, which this does
// not match). It posts the same messages as leetcode-detect.js: a
// "submission" with the judged code, then "accepted" when it passed.
// problemId is NeetCode's slug; the content script maps it to LeetCode's.
// Page scripts can forge these messages, so the content script validates
// them and treats them only as a hint to open the confirm panel.
(() => {
  "use strict";
  if (window.__leetgrinderDetect) return;
  window.__leetgrinderDetect = true;

  const SOURCE = "leetgrinder-detect";
  const SUBMIT = /^\/api\/executeCodeFunctionHttp\/?$/;
  // NeetCode's reply has no submission id; number them here instead.
  let sequence = 0;

  function isSubmit(method, url) {
    if (String(method || "GET").toUpperCase() !== "POST") return false;
    try {
      const u = new URL(url, window.location.href);
      return u.origin === window.location.origin && SUBMIT.test(u.pathname);
    } catch {
      return false;
    }
  }

  function inspect(requestBody, response) {
    let sent;
    try {
      sent = JSON.parse(requestBody).data;
    } catch {
      return;
    }
    if (!sent || typeof sent.problemId !== "string" || typeof sent.rawCode !== "string" || typeof sent.lang !== "string") return;
    const result = response && (response.data || response.result);
    const status = result && result.status && typeof result.status.description === "string" ? result.status.description.slice(0, 64) : "";
    if (!status) return;
    sequence = (sequence + 1) % 1000;
    const id = `${Date.now()}${String(sequence).padStart(3, "0")}`;
    window.postMessage({ source: SOURCE, type: "submission", slug: sent.problemId, submissionId: id, status, lang: sent.lang, code: sent.rawCode }, window.location.origin);
    if (status === "Accepted") {
      window.postMessage({ source: SOURCE, type: "accepted", submissionId: `submission-${id}`, via: "network" }, window.location.origin);
    }
  }

  // Network: wrap XMLHttpRequest (Angular's HttpClient) and fetch without
  // changing their behavior.
  const readJSON = (xhr) => (xhr.responseType === "json" ? xhr.response : JSON.parse(xhr.responseText));

  const originalOpen = XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open = function (method, url, ...rest) {
    this.__leetgrinderSubmit = isSubmit(method, String(url));
    return originalOpen.call(this, method, url, ...rest);
  };

  const originalSend = XMLHttpRequest.prototype.send;
  XMLHttpRequest.prototype.send = function (body) {
    if (this.__leetgrinderSubmit && typeof body === "string") {
      this.addEventListener(
        "load",
        () => {
          try {
            inspect(body, readJSON(this));
          } catch {
            // Not JSON; ignore.
          }
        },
        { once: true },
      );
    }
    return originalSend.apply(this, arguments);
  };

  const originalFetch = window.fetch;
  if (typeof originalFetch === "function") {
    window.fetch = function (...args) {
      const [input, init] = args;
      let text = null;
      try {
        const isRequest = typeof Request === "function" && input instanceof Request;
        const url = isRequest ? input.url : String(input);
        if (isSubmit((init && init.method) || (isRequest ? input.method : "GET"), url)) {
          const body = init && init.body !== undefined ? init.body : null;
          // Read a Request's body from a clone before fetch consumes it.
          text = typeof body === "string" ? Promise.resolve(body) : isRequest && body === null ? input.clone().text() : null;
        }
      } catch {
        text = null;
      }
      const promise = originalFetch.apply(this, args);
      if (text) {
        promise.then((res) => Promise.all([text, res.clone().json()]).then(([sent, body]) => inspect(sent, body)), () => {}).catch(() => {});
      }
      return promise;
    };
  }
})();
