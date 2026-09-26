# DSA leetgrinder

Approved in conversation; the user authorized implementation without further questions.

Build an authenticated Go templ feature at /leetgrinder. Twelve weeks contain 84 numbered sessions, each with three core LeetCode problems. Add four optional problems per week for 252 core and 48 stretch problems, all unique and freely accessible. The audience can program but is rusty on DSA. Sessions advance at the learner's pace via Finish day, regardless of solve outcomes. All sessions remain available for revisiting.

Daily budget: 90 minutes attempting problems, normally three 25-minute attempts plus 15 minutes for debugging; 30 minutes for reading and review. Optional problems use time saved on core problems. A solved count is a count of distinct problems, with independent solves distinguished from assisted solves. Finishing a session does not mark its problems solved.

Progression: fundamentals; binary search; two pointers/windows/prefix sums; linked lists/stacks/recursion; trees; backtracking; graphs; heaps/tries; basic DP; multidimensional DP; greedy/intervals/monotonic stacks; mixed practice.

Write original teaching material and worked examples, using data/algomonster only to understand coverage and prerequisite relationships. Do not publish, embed, copy, or closely paraphrase the source lessons. Research and link primary educational sources with specific reading directions. Every practice assignment links to a LeetCode problem; no in-browser editor.

Pages: overview with continuation and simple counts, daily lesson with assignments and logging, per-problem attempt history with corrections. Use templ components for lessons and every leetgrinder page, Go metadata for assignments, existing Go authentication, PostgreSQL for attempt history and completed sessions. Each attempt records stable problem slug, outcome (solved/struggled/unfinished), minutes, assistance flag, optional notes, timestamp, and revision. Forms preserve input on failure; use idempotent attempt IDs and revision checks for corrections.

No notifications, scheduling, extension, automated spaced repetition, score visualizations, or adjustable timelines. Preserve history so those can be added later.

Build: pin templ runtime and generator; generate during local and Docker builds. Verify 84 sessions, 300 unique problems, assignment counts, links, rendered lessons, auth/origin checks, tracking persistence, retry/correction behavior, distinct solve counts, and session completion independent of results. Include a leetgrinder-specific JSON export for tracking backup without changing the existing workspace archive format.
