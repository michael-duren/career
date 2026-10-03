# Connect the weekly scheduler to Google Calendar

Local schedules work without Google credentials. To enable the optional connection:

1. Enable the Google Calendar API in your Google Cloud project and configure an OAuth web client.
2. Register `${PUBLIC_ORIGIN}/api/scheduler/google/callback` as an authorized redirect URI.
3. Set `SCHEDULER_GOOGLE_CLIENT_ID`, `SCHEDULER_GOOGLE_CLIENT_SECRET`, and `SCHEDULER_SECRET_KEY` on the server. Generate the encryption key with `openssl rand -base64 32` and retain it across restarts.
4. Apply database migrations, build the frontend, and restart the Go service.
5. Open **Weekly Scheduler**, connect Google, and select the calendars that reserve time.

The connection requests OpenID account identity plus `calendar.calendarlist.readonly`, `calendar.events.freebusy`, and `calendar.app.created`. The first calendar scope lists selectable calendars, the second reads busy intervals, and the third manages a dedicated Career Weekly Scheduler calendar. These scopes follow the [Google Calendar authorization reference](https://developers.google.com/workspace/calendar/api/auth) and [FreeBusy reference](https://developers.google.com/workspace/calendar/api/v3/reference/freebusy/query).

The app exports future accepted plans. Editing an exported event in Google does not change the local plan; synchronization restores it. Actual-time corrections stay in Career. Disconnecting removes stored credentials and stops synchronization, but leaves exported events in Google. Reconnecting the same account reuses the destination calendar and stable event identities.

The Go service finalizes elapsed sessions and checks export work every minute. It refreshes availability and reconciles remote changes every five minutes. A failed availability check keeps planning edits unsaved so that they can be retried. Export failures keep local saves and pending synchronization work.

Workspace exports contain scheduler settings, rules, sessions, and historical requirements. They exclude OAuth credentials, Google availability caches, and pending remote jobs. Restore the scheduler encryption key separately if restoring the entire database with encrypted credentials.

Export batches process at most 20 mapped events in a deterministic order with a durable attempt cursor. Failed attempts count toward the limit, and recoverable per-event failures do not block later events. Each mapping stores the desired event fingerprint and generation, plus the last successful fingerprint. Ordinary retries skip unchanged successful writes and retain pending work after a partial failure or interrupted cycle. Existing mappings receive empty progress during migration and are synchronized on the next applicable export or reconciliation cycle. Successful writes retain their event identity; a Google tombstone that rejects reuse receives a persisted replacement identity.

Periodic reconciliation has a durable cursor scoped to the connected account and destination calendar. Each five-minute cycle restores another bounded portion of the accepted export set even when the local plan has not changed. A set larger than 20 events therefore takes multiple periodic cycles to reconcile fully. Forced writes become pending before the provider request; failures retry through the ordinary queue while the reconciliation cursor continues across the full set. Only mapped future events can be removed, and connection changes wait for an active export batch.

Week reads reuse availability for up to five minutes when the cached query covers the requested range and the selected calendars have not changed. Manual and periodic refreshes fetch fresh availability once; the subsequent week read reuses that result. Overlapping refreshes share a fetch across service replicas. A planning mutation always performs its own fresh availability check. Disconnect or calendar changes invalidate in-flight results, and a failed check leaves the proposed edit as a recoverable draft.

Controlled integration tests use the real Google HTTP client against a local test provider and a disposable PostgreSQL database. They cover bounded retries, cancellation, concurrent local edits, account authority, remote deletion/tombstones, refresh request counts, cache expiry/range, and fetch races. Live OAuth consent, Google production quotas and latency, and a connected-account end-to-end run require Google credentials and remain unverified here.
