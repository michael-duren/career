# Leetgrinder content update state

Updated 2026-09-27 (final pass). Work complete against the finish checklist below.

## User requirements

Update all 84 lessons in `internal/leetgrinder`: unique educational pages and references, accessible short first readings, substantial research/white papers, ad-light authoritative sources comparable to OSTEP, and bottom citations for referenced material. Maintain this state file for handoff.

## Implemented

- `model.go`: readings have `Kind` and `Supports`.
- `curriculum.go`: calls `lessonReadings(day)`; problem assignments unchanged.
- `readings.go`: compiled reading catalog, generated from three working JSON manifests, 201 readings across 84 days.
- `lessons.templ`: all 84 lessons end with linked References and source-specific attribution.
- `views.templ`: ordered Reading path; introduction/paper/excerpt labels; mixed-practice readings stay inside review disclosure.
- Tests cover citations, short intros, optional papers, budgets, and unique document URLs after stripping fragments.
- Original first-reading Optional flags preserved because they affect spaced-review scheduling.

## Source of truth while editing

`internal/leetgrinder/sources_01_28.json`, `sources_29_56.json`, `sources_57_84.json` are working manifests, kept alongside the generator rather than deleted: they are the readable record of why each URL was chosen and let future edits regenerate `readings.go` instead of hand-editing generated Go. Run `python3 scripts/leetgrinder/integrate_readings.py` then `gofmt -w internal/leetgrinder/readings.go` after any manifest edit.

## Final verification (this pass)

- `make generate` succeeded.
- `GOCACHE=/tmp/leetgrinder-go-cache go test ./internal/leetgrinder -count=1` passed, including the global duplicate-source URL-uniqueness test.
- `GOCACHE=/tmp/leetgrinder-go-cache go test ./... -count=1` passed in full (sandbox escalation used for local httptest sockets).
- `git diff --check` reported no whitespace errors.
- Live GET audit: `python3 scripts/leetgrinder/check_reading_links.py`. Of 201 readings, 199 return HTTP 200 under strict TLS validation with content verified against each citation's claimed title/topic (not status code alone — this caught paywalls, JSON denials, unrelated PDFs, wrong-paper mismatches, and generic redirect pages in earlier passes). Report: `/tmp/leetgrinder-links.json`; cached documents: `/tmp/leetgrinder-link-check`.
- The remaining 2 (day 69's Colorado dice-recurrence page, day 74's `cse.unl.edu` Allen manuscript) fail Python's strict certificate-chain validation because each server omits an intermediate certificate (`openssl s_client` return code 21, "unable to verify the first certificate"). Both were independently confirmed, via a certificate-insecure fetch, to serve the correct, on-topic document. No independently hosted equivalent was found for either after a genuine search, so both are kept with the caveat recorded in `docs/leetgrinder-review-57-84.md` rather than traded for a weaker or off-topic source with a clean handshake.

## Fixes applied this pass (days 68, 76, 79 in `sources_57_84.json`)

- **Day 68**: the Elzinga–Rahmann–Wang paper has no working open copy anywhere audited (NSYSU mirror times out; ScienceDirect, Ulster, and VU Amsterdam repository pages all resolve only to the same paywalled publisher record). Replaced with Collins's open arXiv paper on counting distinct binary-string subsequences, which gives the same addition-over-cases recurrence; retitled and rescoped.
- **Day 76**: the `www2.hawaii.edu` mirror of the Sitchinava ANSV paper now 404s. Replaced with the same paper's NSF-funded open-access copy (`par.nsf.gov`), confirmed by direct fetch.
- **Day 79**: the third reading kept a live URL but was scoped to an unrelated Minimum Window Substring section. Rescoped to the same page's "At Most K Distinct Characters" section, which is the same atMost(K) technique family as the lesson's `atMost(k) - atMost(k-1)` exact-count reduction.

All other days flagged in earlier passes (57, 59–61, 63–65, 71–73, 78, 80, 84, and the day 4/7/9/10/11/12/14/17/20/22/49/79 duplicate-URL and metadata fixes) were already corrected in the manifests before this pass; this pass re-verified each one live and confirmed the fixes hold. See `docs/leetgrinder-review-57-84.md` for the per-day disposition and `docs/leetgrinder-review-01-28.md` / `docs/leetgrinder-review-29-56.md` for the earlier ranges (no fixes were needed there — only scope notes on what the readings do not directly cover).

## Review agents

Per-day review records exist for all 84 lessons:

- [Lessons 1–28](leetgrinder-review-01-28.md) — all pass; scope notes only.
- [Lessons 29–56](leetgrinder-review-29-56.md) — all pass; scope notes only.
- [Lessons 57–84](leetgrinder-review-57-84.md) — all fixed or pass this pass; two accepted TLS caveats (days 69, 74) noted above.

## Finish checklist

- [x] Resolve source access/accuracy/uniqueness with actual content checks, including PDF titles.
- [x] Reintegrate final manifests; run `make generate`; global content tests; full Go suite with escalation.
- [x] Update `docs/leetgrinder-sources.md` and review docs to reflect final URLs/status.
- [x] Update this state file with final status and exact test outcomes.
- No commits/push requested or made.
