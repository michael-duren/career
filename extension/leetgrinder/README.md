# Leetgrinder browser extension

Logs LeetCode submissions to your Leetgrinder app. When a submission is Accepted, a panel opens on the LeetCode page with the outcome, minutes, and hint use filled in. Nothing is sent until you press **Log attempt**; **Dismiss** sends nothing.

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

After pulling changes, press the reload icon on the extension's card, then reload open LeetCode tabs.

### Firefox

1. Open `about:debugging#/runtime/this-firefox`.
2. Press **Load Temporary Add-on…** and pick `extension/leetgrinder/manifest.json`.
3. Open `about:addons`, choose Leetgrinder, and under **Permissions** make sure access to `leetcode.com` is on (Firefox treats Manifest V3 host access as opt-in).
4. Open **Options** from the same page.

Temporary add-ons are removed when Firefox restarts; load it again afterwards. Your options stay in `storage.local` as long as the add-on id (`leetgrinder@career-strategy.local`) is unchanged.

`web-ext lint` reports three expected warnings: Firefox ignores `background.service_worker` (Chrome uses it; Firefox uses `background.scripts`), and `data_collection_permissions` is newer than the minimum Firefox version (older versions ignore it). The extension sends data only to the app origin you configure.

## How it works

| File | Role |
|---|---|
| `manifest.json` | Declares both `background.service_worker` (Chrome) and `background.scripts` (Firefox). Host access to your app is an optional permission requested at runtime for that origin only. |
| `browser-shim.js` | `ext` = `browser` in Firefox, `chrome` in Chrome. |
| `lib.js` | Pure helpers (slug parsing, timer rounding, outcome inference, validation). Tested with `node --test`. |
| `background.js` | The only code that reads the token and calls the app. Validates who sent each message and what it contains. Keeps per-problem timers in `storage.session`. |
| `leetcode-detect.js` | Runs in LeetCode's page world. Watches `/submissions/detail/<id>/check/` responses for `status_msg: "Accepted"`, with a DOM fallback on the result panel. All LeetCode-specific detection lives here. |
| `content.js` | Tracks the open problem and timer, shows the confirm panel and the 25-minute nudge. |
| `options.html`, `options.js` | App origin, token, and Test connection. |

Behavior:

- **Curriculum only.** On each problem page the extension asks the app whether the slug is in the curriculum. Other problems get no timer, panel, or nudge. If the app cannot be reached, the timer still starts; an Accepted submission then shows the error with **Try again**.
- **Timer.** Starts when a curriculum problem first opens and survives page reloads (it lives in `storage.session`, so a browser restart clears it). It resets after an attempt is logged. It also restarts when the problem has not been open for 30 minutes, or when it is more than 12 hours old, so a later visit (for example a review days later) starts fresh.
- **Assisted.** Opening the problem's Solutions or Editorial tab marks the attempt as assisted.
- **Prefill.** Solved if the timer shows 25 minutes or less, otherwise struggled. Every field is editable.
- **Reviews.** When the problem is one of today's planned reviews, the panel shows a badge and a "Count as today's review" checkbox, checked unless today's review was already logged.
- **Nudge.** After 25 minutes without an Accepted submission, a small panel offers **Log as unfinished** (opens the confirm panel prefilled as unfinished) or **Keep going**. It appears once per problem timer, and never after an Accepted submission on that timer (logged or dismissed). The timer that starts after logging an attempt does not nudge either.
- **Retries.** Each panel has one attempt id. After a network or server error the fields lock and **Retry** resends the identical attempt, so the app never records it twice. While a save is in flight or waiting for Retry, a new Accepted submission does not replace the panel. A 409 means that id was already saved with different values; correct the attempt in the app.

The page world can forge the "Accepted" message. That only opens the panel; sending always needs your click in the extension's closed shadow-DOM panel, and the background worker re-validates every field.

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
- [ ] Open a curriculum problem (for example `https://leetcode.com/problems/two-sum/`). Submit an accepted solution within a few minutes: the panel opens with Solved, the elapsed minutes, and assisted unchecked.
- [ ] Log attempt: the panel says "Logged to Leetgrinder" and closes. The app's problem history shows the attempt with the right values.
- [ ] Reload the problem page mid-attempt: the next Accepted panel's minutes include time before the reload.
- [ ] Close the problem tab, wait over 30 minutes, reopen it and submit: the minutes count from the reopen, and no nudge fires immediately.
- [ ] Open the Solutions or Editorial tab, then submit an accepted solution: assisted is checked.
- [ ] Press **Run** (not Submit) with passing tests: no panel opens.
- [ ] Dismiss: the panel closes and nothing is recorded in the app.
- [ ] Stop the app (or go offline), press Log attempt: an error shows and the fields lock. Restart the app and press Retry: exactly one attempt is recorded.
- [ ] With the app stopped, open a curriculum problem, work a few minutes, start the app, and submit an accepted solution: the panel's minutes include the time before the app came back. With the app still stopped, Accepted shows an error with Try again.
- [ ] While a panel is locked for Retry, submit again: the locked panel stays.
- [ ] Open a problem outside the curriculum and submit an accepted solution: no panel.
- [ ] Keep a curriculum problem open for 25 minutes without submitting: the nudge appears once. **Log as unfinished** opens the panel with Unfinished.
- [ ] Solve a problem in under 25 minutes, then log or dismiss the panel and leave the tab open past the 25-minute mark: no nudge appears.
- [ ] With a problem planned as today's review (see `/leetgrinder/reviews`), the panel shows "Today's review" and the attempt is recorded as a review.
- [ ] Navigate between problems inside LeetCode without a full reload: the panel follows the new problem.
- [ ] Notes containing `<b>html</b>` display as plain text in the app and in the panel.
