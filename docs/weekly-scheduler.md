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
