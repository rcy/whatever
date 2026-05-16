# TODO

## Durable background job processing

### Problem

The classify worker (and enrich worker) run async work in goroutines. This has several gaps:

- If the server restarts mid-classification, in-flight work is lost
- Failed OpenAI calls are printed to stdout and lost — no retry
- The CLI relies on a WaitGroup to avoid exiting before goroutines complete, but a crash still loses work

A naive event-log-as-queue approach was considered (replay `NoteClassificationRequested` and `NoteClassificationFailed` events from a cursor after the projection is fully built) but the interactions between cursor timing, projection ordering, idempotency, goroutine lifetime, and failure recording are subtle and error-prone — easy to get duplicate or missed work at crash boundaries.

### Proposed solution

Investigate and plan adoption of **River** with a **SQLite driver**.

- River handles retries, failure tracking, and job lifecycle correctly by design
- SQLite keeps it self-contained (no new infrastructure)
- The event log stays focused on ordered durable facts; River handles "do this work reliably" separately
- Clean separation of concerns rather than one system doing double duty

### Links

- https://riverqueue.com
- River SQLite driver: investigate availability/maturity
