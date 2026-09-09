# Workflow State

**Status:** Implemented state, 9 September 2026  
**Perspektif:** Semua jenis sambungan

Status node dihitung ulang dari fakta persisted dan dependency graph. `CurrentStage`
adalah proyeksi presentasi dan bukan transition gate.

```mermaid
stateDiagram-v2
  state "Node lifecycle" as NodeLifecycle {
    [*] --> locked: node dibuat
    locked --> available: prerequisite selesai / skipped<br/>dan applicable
    locked --> skipped: keputusan membuat node<br/>tidak applicable
    available --> in_progress: transition start
    available --> completed: transition complete
    in_progress --> completed: transition complete
    completed --> [*]
    skipped --> [*]
  }

  state "Aggregate lifecycle" as AggregateLifecycle {
    [*] --> aggregate_in_progress: permohonan dibuat
    aggregate_in_progress --> aggregate_returned: nps_delegation=returned
    aggregate_in_progress --> aggregate_completed: selesai=completed
    aggregate_returned --> [*]
    aggregate_completed --> [*]
  }
```

`returned` dan `completed` bersifat terminal. Node `completed` atau `skipped` dapat
memenuhi prerequisite; skip hanya sah jika condition node memang tidak applicable.
