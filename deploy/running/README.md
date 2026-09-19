# Running notes deployment and verification

The Go application defaults to transcription disabled. Recording, durable local queueing,
uploading and editing still work. Only `WHISPER_URL` receives audio; no browser speech
API or paid/third-party speech service is used. Choose an internal endpoint.

1. Build `docker build -f deploy/running/Dockerfile -t career-whisper:v1.7.6 .` and
   import the image onto the homelab nodes with the normal infrastructure workflow.
   Pin the resulting image digest in production. This CPU image includes ffmpeg.
2. Copy `whisper.yaml` into the homelab infrastructure repository. Check its namespace,
   storage class, node architecture and app pod labels. Provision the `whisper-models`
   PVC with `ggml-base.en.bin` obtained using the upstream model download procedure
   (an administrator downloads the model, never user audio). The model must be readable
   by UID 65532. There is no public Ingress or runtime model downloader. The network
   policy denies all whisper egress and allows only application pod ingress.
3. Run the application `migrate` command before rolling out the new app. Migration 002
   stores clips in PostgreSQL BYTEA, so ordinary database backups include audio.
   JSON export contains running note text/metadata, without binary clips; JSON export
   is not an audio backup. Retain PostgreSQL backups for full restore.
4. Set `RUNNING_TRANSCRIBE_ENABLED=true`, `WHISPER_URL=http://whisper:8080` and
   `WHISPER_MODEL=ggml-base.en` in the Go Deployment. The model setting records the
   server's configured model, so keep it consistent with the whisper model mount/args.
   Model `small.en` can be evaluated by changing both. Roll back to disabled at any time.
5. Inspect `running_transcribe_duration_seconds` and `running_transcribe_failures_total`
   using the existing OTel metrics pipeline. Upload/transcription spans are emitted
   through the global tracer (a trace provider must be configured to export spans).

The worker sends multipart `file` to `/inference` with `response_format=json`.
`--convert` converts browser audio to whisper's WAV input using ffmpeg. See the
[upstream server documentation](https://github.com/ggml-org/whisper.cpp/tree/master/examples/server).
There is one in-flight job per application process; database claims use SKIP LOCKED.
Unavailable/5xx servers leave clips pending without consuming the three content-error
attempts. Retry delay is 60/90/120 seconds; interrupted claims expire after ten minutes.
A four-minute inference timeout prevents a worker outliving its lease.

# Phone and homelab release checks (not yet verified)

Actual phone recording/lock behavior, microphone routing, wind accuracy and homelab
CPU latency require the user's hardware; automated fake-server tests cannot establish
these properties. The supplied deployment image has not been built/deployed here.

- Visit `/running` online and reload after the service worker activates, then verify
  it opens in airplane mode. Only its data-free shell and public hashed assets are
  cached; APIs/audio remain network-only. Cached shell data and queued audio persist
  on the device until upload or browser storage removal. Do not clear site data or
  uninstall the PWA with pending clips. User edits continue to use existing drafts.
- Record three short clips with no signal. Stop each before pocketing the phone.
  Confirm the waiting count is three, close/reopen the page, restore Wi-Fi and verify
  three segments exactly once in one run. A lost response must not duplicate a clip.
- Try intentional screen lock, pocketing and Bluetooth headphones; record the device,
  browser, MIME type and whether the OS suspended capture. Wake Lock is best effort.
  Import a native voice memo (M4A/MP3/WAV/etc., readable duration, <=30 min, <=15 MiB)
  if background recording fails. Import uses the file modification time for grouping.
- Time a one-minute wind/breath clip from upload to done. The acceptance target is
  under two minutes on homelab CPU; not established by the test suite. Assess base.en
  accuracy before allocating more resources or switching to small.en.
- Stop whisper, upload a clip, confirm pending while the rest of the app works, then
  restore whisper and confirm completion. Edit a run while another clip completes:
  saving a stale revision must preserve the draft and show a conflict.

Run `TEST_DATABASE_URL=... go test -race ./...` against disposable PostgreSQL for
schema-isolated claim/upload/retry/API/fake-whisper coverage. `npm test` exercises
local queue persistence/replay, search separation and explicit context opt-in.
