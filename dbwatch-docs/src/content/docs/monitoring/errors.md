---
title: Errors and events
description: One timeline of warnings and worse from the other panels, plus deadlocks. The status is the latest event, not a running worst.
---

Errors / events has no query of its own. Whenever another panel records an event at warning or above, a copy is appended here. Deadlocks are included too: `pg_stat_database.deadlocks` is a counter, and an increase is logged here even though deadlocks do not have their own panel.

You get one list of connection pressure, cache degradation, lock waits, long transactions, extension failures, and deadlocks, instead of checking six panels to find what changed.

The status dot is the severity of the most recent event. A warning that already cleared does not stay latched. There is no reliable signal that a past log line has been "resolved", so the panel shows what just happened.

Select the panel and the bottom log switches to this merged history. Enter opens the full detail. `c` copies it.
