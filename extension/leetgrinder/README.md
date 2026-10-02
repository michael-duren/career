# Leetgrinder browser extension

Logs LeetCode and NeetCode submissions to your Leetgrinder app and shows today's goal. The toolbar badge counts what is left of today's goal, the popup lists today's progress and reviews, and a small banner on each problem page says whether it is new, due, or flagged. When a submission is Accepted, a panel opens on the problem page with the outcome, minutes, and hint use filled in. You state the time and space complexity, and the code LeetCode judged is attached unless you untick it. Nothing is sent until you press **Log attempt**; **Dismiss** sends nothing.

Manifest V3, plain JavaScript, no build step. One `manifest.json` works in Chrome 121+ and Firefox 128+. It is loaded unpacked; it is not published to any store.

## Set up

1. In the app, open **Leetgrinder → Settings → Browser extension tokens**, name a token (for example "Firefox laptop"), and press **Create token**. Copy the `lg_…` value now; the app shows it only once and stores only its SHA-256 hash.
2. Load the extension (below).
3. Open the extension's options, enter the app origin (for example `https://career.example.com`) and the token, and press **Save**. The browser asks for access to that one origin; allow it. Only `https://` origins are accepted, plus `http://localhost` and `http://127.0.0.1` for development, so the token never travels unencrypted.
4. Press **Test connection**. "Connected" means the token works.

To stop using a token, revoke it on the same settings page. Revocation takes effect immediately.

### Chrome (or Chromium, Edge, Brave)

1. Open `chrome://extensions` and turn on **Developer mode**.
2. Press **Load unpacked** and pick this `extension/leetgrinder` directory.
3. Open the extension's **Details → Extension options**.

After pulling changes, press the reload icon on the extension's card, then reload open LeetCode and NeetCode tabs.

### Firefox

1. Open `about:debugging#/runtime/this-firefox`.
2. Press **Load Temporary Add-on…** and pick `extension/leetgrinder/manifest.json`.
3. Open `about:addons`, choose Leetgrinder, and under **Permissions** make sure access to `leetcode.com` and `neetcode.io` is on (Firefox treats Manifest V3 host access as opt-in).
4. Open **Options** from the same page.

Temporary add-ons are removed when Firefox restarts; load it again afterwards. Your options stay in `storage.local` as long as the add-on id (`leetgrinder@career-strategy.local`) is unchanged.

`web-ext lint` reports three expected warnings: Firefox ignores `background.service_worker` (Chrome uses it; Firefox uses `background.scripts`), and `data_collection_permissions` is newer than the minimum Firefox version (older versions ignore it). The extension sends data only to the app origin you configure.

## How it works

| File | Role |
|---|---|
| `manifest.json` | Declares both `background.service_worker` (Chrome) and `background.scripts` (Firefox). Host access to your app is an optional permission requested at runtime for that origin only. |
| `browser-shim.js` | `ext` = `browser` in Firefox, `chrome` in Chrome. |
| `lib.js` | Pure helpers (slug parsing, timer rounding, outcome inference, complexity normalisation, message and payload validation). Tested with `node --test`. |
| `background.js` | The only code that reads the token and calls the app. Validates who sent each message and what it contains. Keeps per-problem timers in `storage.session`, and keeps the toolbar badge current from `GET /api/leetgrinder/today`. |
| `leetcode-detect.js` | Runs in LeetCode's page world. Records the JSON body of `POST /problems/<slug>/submit/` (`lang`, `typed_code`) with the `submission_id` from its response, and watches `/submissions/detail/<id>/check/` responses. When a check finishes it posts `{slug, submissionId, status, lang, code}`, then an Accepted message if `status_msg` is `"Accepted"`. A DOM fallback on the result panel covers Accepted without code. All LeetCode-specific detection lives here. |
| `neetcode-detect.js` | Runs in NeetCode's page world. Watches `POST /api/executeCodeFunctionHttp` (Submit; Run uses `runCodeFunctionHttp` and is ignored): the request's `data` has `problemId` (NeetCode's slug), `rawCode` and `lang`, and the response's `data.status.description` is the verdict (a pass also needs `correct_test_case_count` to equal `test_case_count` when both are present). It posts the same messages as `leetcode-detect.js`, numbering submissions itself since NeetCode returns no id. All NeetCode-specific detection lives here. |
| `neetcode-slugs.js` | Generated map from NeetCode slug to `[LeetCode slug, number, title, difficulty]`, taken from the problem list neetcode.io ships in its main bundle (the `ncLink` → `link` pairs NeetCode itself uses). Regenerate with `node scripts/update-neetcode-slugs.js`. |
| `content.js` | Tracks the open problem, timer heartbeat and input activity, validates page-world messages, and shows the confirm panel and the 25-minute nudge. It also samples page visibility and input for active time. |
| `options.html`, `options.js` | App origin, token, and Test connection. |
| `popup.html`, `popup.js`, `popup.css` | The toolbar popup: today's progress, streak, review picks, the due list (top 10), and a dashboard link. Read-only. |

Behavior:

- **Any problem.** Every `leetcode.com/problems/<slug>/` page gets a timer, the confirm panel, and the nudge. On each problem page the extension asks the app what it knows about the slug (new, due for review, or not due). If the app cannot be reached, the timer still starts; an Accepted submission then shows the error with **Try again**.
- **NeetCode.** `neetcode.io/problems/<slug>` pages behave the same, keyed by the LeetCode slug the NeetCode problem mirrors (`two-integer-sum` is `two-sum`), so a problem shares its timer and history across both sites. NeetCode-only problems (courses, problems missing from `neetcode-slugs.js`) are ignored; regenerate the table when NeetCode adds problems. The title, number and difficulty come from the table; the app keeps the problem queued for its LeetCode fetcher, which adds the topics. The Solution tab marks the attempt assisted. Popup links still open LeetCode.
- **Problem details.** When the app does not know a problem's title or topics yet, the content script reads them same-origin from `https://leetcode.com/graphql` (`question(titleSlug) { questionFrontendId title difficulty topicTags { slug name } }`), sends them to `PUT /api/leetgrinder/problem/<slug>`, and attaches them to logged attempts. Nothing else is read from LeetCode's API.
- **Timer.** Starts when a problem first opens and survives page reloads (it lives in `storage.session`, so a browser restart clears it). It resets after an attempt is logged. It also restarts when the problem has not been open for 30 minutes, or when it is more than 12 hours old, so a later visit (for example a review days later) starts fresh.
- **Active time.** The timer keeps two clocks: how long the problem has been open, and active time. Active time only advances while the tab is visible and you pressed a key, clicked, moved the mouse or scrolled within the last 10 minutes. The 10-minute window covers thinking time without typing; the tradeoff is that a break still counts for up to 10 minutes after your last input. The content script sends a visibility and last-input sample with each 30-second heartbeat (and on tab hide or show). The background worker is still the only writer. It keeps a previous-sample time per tab, credits at most 90 seconds per sample (the tick sends its heartbeat before waiting on the app lookup, so a slow lookup does not delay it), counts the union of what tabs report (a hidden twin tab or a second visible tab never changes the total), and never lets active time exceed the time since the timer started. A small backward clock step (under 90 seconds, such as NTP slew) only clamps the stored marks. A larger jump keeps the active time already earned, credits nothing for that one sample, and shifts the open time so "open M min" and the 12-hour max age continue as if the gap never happened. A forward step of more than 30 minutes expires the timer like sleep, and the next visit starts fresh. A reload or navigation earns no active time until you next press, click, move or scroll. Prefill, the outcome guess and the nudge use active time. The confirm panel shows "Active N min (open M min)" when the two differ.
- **Assisted.** Opening the problem's Solutions or Editorial tab marks the attempt as assisted.
- **Prefill.** Minutes are the active minutes. Solved if that is 25 or less, otherwise struggled. Every field is editable.
- **Complexity.** Time and Space each offer the common classes (`O(1)` to `O(n!)`) or **Other…**, which shows a text field for values such as `O(m·n)` or `O(V + E)`. Other values must start with `O(`, end with `)`, and fit in 40 characters; `n^2`, `nlogn` and `logn` are rewritten as `n²`, `n log n` and `log n`. Both are required for solved and struggled attempts and optional for unfinished ones; the panel refuses to send without them. The same rules run in the app.
- **Code.** When the judged submission's code was captured, the panel shows "Code captured (Python3, 1.2 KB)" with a checkbox, ticked by default, to include it. The Accepted panel attaches the code of the submission it reports; the unfinished panel attaches the latest submission on the problem, whatever its result. Code over 64 KiB is never captured, and code whose request would exceed the app's 96 KiB body limit is left out (the panel says so after saving). Code goes only to the configured app origin, through the background worker.
- **Kind.** The panel's badge says how the app will count the attempt today: "New", "Review", or "Practice" (from `todayKind` in the status call). The app decides; after saving, the panel says "Logged to Leetgrinder as Review." with the kind the app reported. The extension still sends `isReview: false` for older apps, which the app ignores.
- **Badge.** The background worker reads `GET /api/leetgrinder/today` on startup, after every logged attempt, when the options change, and every 15 minutes (`chrome.alarms`, hence the `alarms` permission). The badge shows how many new problems and reviews are left, a green ✓ when the goal is met, or a grey `?` (with a "not connected" title) when the extension is not set up or the app cannot be reached.
- **Popup.** Clicking the toolbar icon shows today's goal progress (new x/n, review x/n, bonus), the streak, today's review picks and the due list (top 10) with LeetCode links, and a link to the dashboard. It never writes.
- **Banner.** A pill under the problem title (or fixed top-right when the title cannot be found), isolated in a closed shadow root: "New", "Review due · recall 62%", "Today's review", "Flagged: time complexity judged wrong", or "Reviewed 3 d ago · next due Oct 14", with the last attempt ("Struggled · 32 min · O(n log n)/O(n)"). Clicking it opens the problem's history in the app. It is hidden when the extension is not set up or the app cannot be reached, and updates after logging.
- **Nudge.** After 25 active minutes without an Accepted submission, a small panel offers **Log as unfinished** (opens the confirm panel prefilled as unfinished) or **Keep going**. It appears once per problem timer, and never after an Accepted submission on that timer (logged or dismissed). The timer that starts after logging an attempt does not nudge either.
- **Retries.** Each panel has one attempt id. After a network or server error the fields lock and **Retry** resends the identical attempt, so the app never records it twice. While a save is in flight or waiting for Retry, a new Accepted submission does not replace the panel. While the panel is waiting for Retry, Escape does nothing and Dismiss asks before discarding the attempt. A 409 means that id was already saved with different values; the fields unlock with what you typed and a new id, so you can check the history or save it as a new attempt.

The page world can forge both messages. The content script accepts them only from the same window and origin, and checks a submission message's exact keys, types, and sizes (numeric submission id, status up to 64 characters, a LeetCode-style language slug up to 32 characters, code up to 64 KiB) and that it names the problem on screen. A forged message can only open the panel or offer code that the panel shows with its size; sending always needs your click in the extension's closed shadow-DOM panel, and the background worker re-validates every field. The app escapes code when it displays it.

## Icons

`icons/icon.svg` is the source. Chrome needs PNG manifest icons, so regenerate them after editing it:

```sh
cd extension/leetgrinder
for s in 16 32 48 128; do rsvg-convert -w $s -h $s icons/icon.svg -o icons/icon-$s.png; done
```

## Tests

```sh
cd extension/leetgrinder
node --test
```

## Manual test checklist

Run through this after loading the extension in each browser, with the app running and a token configured.

- [ ] Options: saving an `http://` non-localhost origin is rejected; saving a malformed token is rejected.
- [ ] Options: Save prompts for access to the app origin only; Test connection says "Connected".
- [ ] Options: Test connection with a revoked token reports that the token was rejected.
- [ ] Open a problem (for example `https://leetcode.com/problems/two-sum/`). Submit an accepted solution within a few minutes: the panel opens with Solved, the elapsed minutes, and assisted unchecked.
- [ ] Log attempt without choosing Time or Space: the panel refuses with a message. Choose both (try **Other…** with `O(nlogn)`) and log: the panel says "Logged to Leetgrinder" and closes. The app's problem history shows the attempt with the right values, `O(n log n)`, and the submitted code under "Submitted code".
- [ ] The Accepted panel shows "Code captured (<language>, <size>)". Untick it and log: the history shows no code.
- [ ] Choose **Took a simpler approach for time** and tick **Review this again soon**, then log: the history shows both, and the next day the banner reads "Flagged: simpler approach taken" (or "Today's review" when it is one of the day's picks).
- [ ] Submit a wrong answer, then wait for the 25-minute nudge and **Log as unfinished**: the panel offers the wrong answer's code and complexity is optional.
- [ ] Active time: open a problem, work for a couple of minutes, then switch to another tab for 10 minutes and come back. Submit an accepted solution: the panel's minutes exclude the hidden time and it shows "Active N min (open M min)".
- [ ] Active time: open a problem and leave it visible without touching the keyboard or mouse for over 10 minutes before submitting: the minutes stop growing after about 10 idle minutes, and Accepted does not prefill Struggled just because the tab stayed open.
- [ ] Active time: open the same problem in two tabs, work in one while the other stays in the background: the active minutes match the time you spent working, not half of it or double.
- [ ] Active time: after the above, type or scroll again: active time resumes. When active and open minutes match, no "Active ... (open ...)" line shows.
- [ ] Reload the problem page mid-attempt: the next Accepted panel's minutes include time before the reload.
- [ ] Close the problem tab, wait over 30 minutes, reopen it and submit: the minutes count from the reopen, and no nudge fires immediately.
- [ ] Open the Solutions or Editorial tab, then submit an accepted solution: assisted is checked.
- [ ] Press **Run** (not Submit) with passing tests: no panel opens.
- [ ] Dismiss: the panel closes and nothing is recorded in the app.
- [ ] Stop the app (or go offline), press Log attempt: an error shows and the fields lock. Restart the app and press Retry: exactly one attempt is recorded.
- [ ] With the app stopped, open a problem, work a few minutes, start the app, and submit an accepted solution: the panel's minutes include the time before the app came back. With the app still stopped, Accepted shows an error with Try again.
- [ ] While a panel is locked for Retry, submit again: the locked panel stays. Press Escape: nothing happens. Press Dismiss: it asks before discarding.
- [ ] Log an attempt, edit it in the app, then Retry the same panel after a forced failure (or reuse its id): a 409 shows "Saved earlier with different values...", the fields unlock with what you typed, and pressing Log attempt saves it under a new id.
- [ ] Open a problem you have never logged (for example `https://leetcode.com/problems/design-hit-counter/`) and submit an accepted solution: the panel opens with the "New" badge. After logging, the app's problem page shows its title, number, difficulty, and topics.
- [ ] Keep a problem open and actively working (typing or scrolling) for 25 minutes without submitting: the nudge appears once. **Log as unfinished** opens the panel with Unfinished.
- [ ] Solve a problem in under 25 minutes, then log or dismiss the panel and leave the tab open past the 25-minute mark: no nudge appears.
- [ ] With a problem picked as today's review (see `/leetgrinder/reviews`), the banner says "Today's review", the panel badge says "Review", and after logging the panel says "Logged to Leetgrinder as Review." Re-solving a problem that is not due shows "Practice".
- [ ] The toolbar badge shows the number left for today's goal; logging a new problem lowers it within a few seconds; meeting the goal shows a green ✓. Revoke the token (or stop the app) and wait for the next refresh, or reload the extension: the badge shows a grey `?`.
- [ ] From the browser console of a LeetCode page, `chrome.runtime.sendMessage({type: "today"})` is refused ("Unexpected sender."); from the popup it works. `open-history` with an invalid slug is refused.
- [ ] Revoke the app origin's site access in the browser's extension settings: the badge turns to a grey `?` right away.
- [ ] The popup lists today's progress, streak, picks and due problems; each link opens LeetCode; "Open the dashboard" opens the app.
- [ ] The banner appears under the title of any problem, says "New" for a problem never logged, and after logging an attempt switches to "Reviewed today · next due …" (or "Today's review · done" for today's pick) with the attempt summary. Clicking it opens the problem's history page in the app. With no token configured, no banner appears.
- [ ] Navigate between problems inside LeetCode without a full reload: the panel follows the new problem.
- [ ] Notes containing `<b>html</b>` display as plain text in the app and in the panel.
- [ ] NeetCode: open `https://neetcode.io/problems/two-integer-sum/question` signed in. The banner shows Two Sum's status (the same one as on `leetcode.com/problems/two-sum/`). Submit an accepted solution: the panel opens with the code captured (Python, …) and logging records it under `two-sum` in the app.
- [ ] NeetCode: press **Run** with passing tests: no panel opens. Submit a wrong answer, then wait for the nudge: **Log as unfinished** offers that code.
- [ ] NeetCode: open the Solution tab, then submit an accepted solution: assisted is checked.
- [ ] NeetCode: open a course or NeetCode-only problem: no banner, no timer.
