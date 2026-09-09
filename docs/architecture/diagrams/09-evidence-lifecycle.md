# Evidence Lifecycle

**Status:** Implemented state, 9 September 2026  
**Perspektif:** Dokumen private dan association ke workflow node

```mermaid
flowchart TD
  upload["POST /api/documents<br/>authenticated uploader"] --> validate{"Validate request,<br/>type, filename, ≤10 MiB"}
  validate -->|invalid| reject400["Reject 400"]
  validate -->|valid| sniff{"Read bounded bytes<br/>detected MIME = declared MIME?"}
  sniff -->|tidak| rejecttype["Reject invalid file"]
  sniff -->|ya| configured{"CLAMAV_ADDRESS<br/>configured?"}
  configured -->|ya| scan["Synchronous ClamAV scan"]
  configured -->|tidak| notscanned["scan_status = not_scanned"]
  scan -->|infected| malware["Reject malware"]
  scan -->|error / unknown| scanfail["Reject: fail closed"]
  scan -->|clean| clean["scan_status = clean"]
  clean --> object["Put opaque key in private Garage/S3"]
  notscanned --> object
  object --> revision{"supersedes_document_id?"}
  revision -->|tidak| persist["Persist revision 1<br/>unattached"]
  revision -->|ya; same owner/type,<br/>old still unattached| supersede["Transaction: create next revision<br/>mark old superseded"]
  revision -->|invalid predecessor| compensate["Delete new object<br/>reject revision"]
  persist --> unattached["Unattached upload"]
  supersede --> unattached
  unattached -->|activity submission| attach{"Atomic attachment checks"}
  attach -->|missing / wrong uploader /<br/>superseded / other request / duplicate| rollback["Rollback activity + evidence"]
  attach -->|valid| bound["Bind document to permohonan<br/>create DocumentEvidence per node"]
  bound --> immutable["Attached evidence immutable<br/>one file may evidence multiple nodes"]
  immutable --> download["Authorized read guard<br/>15-minute presigned download"]
  unattached -->|older than configured TTL| cleanup["cleanup-orphan-documents"]
  cleanup --> deleteobject["Delete private object"] --> deleterow["Delete row if still unattached"]

  classDef reject fill:#fee2e2,stroke:#b91c1c,color:#7f1d1d
  classDef stored fill:#dcfce7,stroke:#15803d,color:#14532d
  class reject400,rejecttype,malware,scanfail,rollback reject
  class persist,supersede,bound,immutable,download stored
```

Object key, original filename, customer data, dan presigned URL tidak ditulis ke
activity log. Jika persistence upload gagal setelah object dibuat, service mencoba
menghapus object sebagai kompensasi.
