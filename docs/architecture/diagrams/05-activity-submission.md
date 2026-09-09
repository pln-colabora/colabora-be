# Activity Submission Sequence

**Status:** Implemented state, 9 September 2026  
**Perspektif:** Semua endpoint penyelesaian activity

Dokumen sudah di-upload dan dipindai sebelum submission. Penyelesaian node, attachment
evidence, aggregate projection, dan audit log berada dalam satu transaksi database.

```mermaid
sequenceDiagram
  autonumber
  actor Client
  participant C as Gin Controller
  participant S as Permohonan Service
  participant DB as PostgreSQL / Repository
  participant W as Workflow + RBAC
  participant E as Document Evidence Repository
  participant L as Activity Log

  Note over Client,E: document_ids menunjuk upload unattached milik caller
  Client->>C: POST activity payload + document_ids
  C->>C: Bind dan validate request
  alt Payload invalid
    C-->>Client: 400 Bad Request
  else Payload valid
    C->>S: Submit(ctx, id, user_id, request)
    S->>DB: BEGIN
    S->>DB: SELECT permohonan + nodes FOR UPDATE
    DB-->>S: Aggregate snapshot + actor
    S->>W: Evaluate + authorize exact node
    alt Wrong owner
      W-->>S: forbidden
      S->>DB: ROLLBACK
      S-->>C: authorization error
      C-->>Client: 403 Forbidden
    else Node locked, skipped, completed, or terminal
      W-->>S: not actionable
      S->>DB: ROLLBACK
      S-->>C: state conflict
      C-->>Client: 409 Conflict
    else Actionable
      S->>W: Transition node(s) to completed
      W-->>S: Recomputed nodes + aggregate projection
      S->>E: Attach every document to every submitted node
      alt Missing, superseded, mismatched, or attached document
        E-->>S: attachment error
        S->>DB: ROLLBACK
        S-->>C: evidence error
        C-->>Client: 404 or 409
      else Evidence attached
        S->>DB: Persist payload, states, decisions, stage/status
        S->>L: Insert node_completed and derived node_skipped events
        S->>DB: COMMIT
        S-->>C: Updated aggregate + available_actions
        C-->>Client: 200 Success
      end
    end
  end
```

Bundled endpoints menjalankan transisi node dalam dependency order dan hanya menyimpan
hasil akhir setelah seluruh validasi berhasil.
