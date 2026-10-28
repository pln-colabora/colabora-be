# Evidence Lifecycle

**Status:** Implementasi branch dev, 26 Oktober 2026
**Perspektif:** Dokumen private dan association ke workflow node

```mermaid
flowchart TD
  register["POST /api/auth/register<br/>multipart user + document"] --> accountvalidate{"Validate registration<br/>fields and file"}
  accountvalidate -->|invalid| accountreject["Reject registration"]
  accountvalidate -->|valid| accountstore["Store private object<br/>and create user/document/account_documents<br/>in one DB transaction"]
  accountstore -->|commit failure| accountcleanup["Delete object as compensation"]
  accountstore --> accountpending["is_verified = false<br/>login denied until account-manager verification"]
  upload["POST /api/documents<br/>authenticated uploader"] --> validate{"Validate request,<br/>type, filename, ≤30 MiB"}
  validate -->|invalid| reject400["Reject 400"]
  validate -->|valid| sniff{"Read bounded bytes<br/>detected MIME = declared MIME?<br/>PDF/JPEG/PNG/XLS/XLSX"}
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
  bound --> immutable["File terlampir tetap tersimpan<br/>satu file dapat menjadi evidence beberapa node"]
  immutable --> revise{"Revisi survei atau lampiran<br/>upload WO Konstruksi?"}
  revise -->|owner, completed, revision dan evidence valid| snapshot["Simpan payload dan pilihan lama<br/>ActivityRevision"]
  snapshot --> select["Pilih evidence terbaru<br/>ubah is_current + revision + audit secara atomik"]
  select --> current["Pilihan evidence terbaru<br/>completion dan SLA tetap"]
  revise -->|revision usang / evidence tidak valid| rollback
  current --> download
  immutable --> download["Authorized read guard<br/>backend-streamed document"]
  unattached -->|older than configured TTL| cleanup["cleanup-orphan-documents"]
  cleanup --> deleteobject["Delete private object"] --> deleterow["Delete row if still unattached"]
  accountpending -. excluded from orphan cleanup .-> deleterow

  classDef reject fill:#fee2e2,stroke:#b91c1c,color:#7f1d1d
  classDef stored fill:#dcfce7,stroke:#15803d,color:#14532d
  class reject400,rejecttype,malware,scanfail,rollback reject
  class persist,supersede,bound,immutable,download stored
```

Object key, original filename, customer data, dan storage credential tidak ditulis ke
activity log. Jika persistence upload gagal setelah object dibuat, service mencoba
menghapus object sebagai kompensasi.

Revisi attachment berlaku hanya pada survei dan lampiran upload WO Konstruksi,
termasuk setelah aggregate completed/returned. File dan evidence historis tidak
dihapus. PDF WO generated serta association node lain tidak berubah; superseding
upload unattached berbeda dari memilih ulang evidence melalui endpoint revisi.

Survei tetap memerlukan minimal satu evidence. Kategori izin pemilik tanah,
perangkat desa, atau gabungan keduanya tersedia dan opsional. Batas per file
30 MiB; XLSX diterima dengan pemeriksaan isi/MIME, termasuk untuk RAB/KKO/KKF.
