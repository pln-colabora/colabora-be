# Data Model

**Status:** Implementasi branch dev, 26 Oktober 2026
**Perspektif:** Entity workflow, evidence, assignment, SLA, dan audit

```mermaid
erDiagram
  USER {
    uuid id PK
    string role
    string unit
    string email UK
    boolean is_verified
  }
  PERMOHONAN {
    uuid id PK
    string no_permohonan UK
    string jenis_sambungan
    string ulp_unit
    date tanggal_pembayaran
    bigint nominal_pembayaran
    string sla_basis
    string energize_status
    boolean legacy_priority_exempt
    smallint current_stage
    string status
    boolean kebutuhan_tiang
    string nps_delegation_status
    boolean perlu_pdkb
    uuid created_by FK
  }
  PERMOHONAN_ACTIVITY {
    uuid id PK
    uuid permohonan_id FK
    string workflow_node UK
    smallint activity_number
    smallint stage_number
    string status
    date sla_deadline
    string sla_basis
    int revision
    jsonb payload
    uuid completed_by FK
  }
  DOCUMENT {
    uuid id PK
    uuid permohonan_id FK
    uuid uploaded_by FK
    string type
    string mime_type
    bigint size_bytes
    string checksum_sha256
    string scan_status
    smallint revision
    uuid supersedes_id FK
    uuid superseded_by_id FK
  }
  DOCUMENT_EVIDENCE {
    uuid id PK
    uuid document_id FK
    uuid permohonan_id FK
    string workflow_node FK
    uuid attached_by FK
    boolean is_current
  }
  ACTIVITY_REVISION {
    uuid id PK
    uuid permohonan_id UK
    string workflow_node UK
    int revision UK
    jsonb payload
    jsonb document_ids
    uuid recorded_by
    timestamp created_at
  }
  PDKB_SCHEDULE {
    uuid id PK
    date schedule_date UK
    string title
    uuid permohonan_id UK
    uuid created_by
    uuid updated_by
  }
  ACCOUNT_DOCUMENT {
    uuid id PK
    uuid user_id UK,FK
    uuid document_id UK,FK
    string document_type
  }
  VENDOR_ASSIGNMENT {
    uuid permohonan_id PK,FK
    string vendor_role PK
    uuid vendor_id FK
    uuid assigned_by FK
  }
  ACTIVITY_LOG {
    uuid id PK
    uuid permohonan_id FK
    string workflow_node
    smallint activity_number
    uuid actor FK
    string action
  }
  SLA_RULE {
    uuid id PK
    smallint activity_number UK
    string jenis_sambungan UK
    smallint offset_days
    string reference_point
  }

  PERMOHONAN ||--o{ ACTIVITY_REVISION : revision_history
  PERMOHONAN o|--o| PDKB_SCHEDULE : books
  USER ||--o{ ACTIVITY_REVISION : records
  USER ||--o{ PDKB_SCHEDULE : manages
  USER ||--o{ PERMOHONAN : creates
  PERMOHONAN ||--|{ PERMOHONAN_ACTIVITY : contains
  USER o|--o{ PERMOHONAN_ACTIVITY : completes
  USER ||--o{ DOCUMENT : uploads
  USER ||--o| ACCOUNT_DOCUMENT : verifies
  DOCUMENT ||--o| ACCOUNT_DOCUMENT : submitted_for
  PERMOHONAN o|--o{ DOCUMENT : binds
  DOCUMENT ||--o{ DOCUMENT_EVIDENCE : classified_as
  PERMOHONAN ||--o{ DOCUMENT_EVIDENCE : owns
  PERMOHONAN ||--o{ VENDOR_ASSIGNMENT : grants
  USER ||--o{ VENDOR_ASSIGNMENT : vendor_or_assigner
  PERMOHONAN ||--o{ ACTIVITY_LOG : audits
  USER ||--o{ ACTIVITY_LOG : acts
  DOCUMENT o|--o| DOCUMENT : supersedes
```

`SLA_RULE` dipilih secara logis memakai `(activity_number, jenis_sambungan)`; tidak ada
foreign key langsung dari workflow node. Constraint evidence memastikan dokumen dan
node berasal dari permohonan yang sama. Satu dokumen dapat menjadi evidence untuk
beberapa node melalui beberapa row `DOCUMENT_EVIDENCE`.

`ACTIVITY_REVISION` menyimpan snapshot payload dan ID dokumen lama, bukan salinan
file. Identitas uniknya `(permohonan_id, workflow_node, revision)`. Relasi pada
entity revisi/jadwal menunjukkan asosiasi logis; tidak menambahkan foreign key
fisik yang tidak dideklarasikan entity. Evidence lama tetap tersimpan dengan
`is_current=false`; file generated tetap pada association semula.

`PDKB_SCHEDULE.schedule_date` unik untuk satu tim/satu tanggal. `permohonan_id`
null berarti kegiatan manual; nilai terisi berarti booking WO yang terkunci.
`payment_workdays_v1` membedakan SLA pembayaran/hari kerja dari `calendar`
historis; migrasi tidak mengganti deadline lama.
