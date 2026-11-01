# API Action Map

**Status:** Implementasi branch dev, 26 Oktober 2026
**Perspektif:** Authenticated write endpoints

```mermaid
flowchart TB
  classDef create fill:#dbeafe,stroke:#1d4ed8,color:#1e3a8a
  classDef evidence fill:#dcfce7,stroke:#15803d,color:#14532d
  classDef assignment fill:#fef3c7,stroke:#b45309,color:#78350f

  subgraph ENTRY["Entry"]
    create_req["POST /api/permohonan<br/>permohonan<br/>PP JTR/JTM · NPS PLG TM<br/>Evidence: tidak<br/>Hasil: node awal completed"]
  end
  subgraph S2S3["Survey dan planning"]
    survey["POST /:id/survei<br/>survei<br/>Teknik · Perencanaan<br/>Evidence: wajib"]
    rab["POST /:id/rab-kko-kkf<br/>rab_kko_kkf + kebutuhan_tiang<br/>Teknik · Perencanaan<br/>Evidence: wajib"]
    priority["POST /:id/verifikasi-prioritas<br/>verifikasi_prioritas · Perencanaan semua jenis<br/>Tanpa evidence · catatan opsional"]
    expansion["POST /:id/permohonan-perluasan<br/>permohonan_perluasan + nps_delegation<br/>NPS<br/>Evidence: wajib"]
  end
  subgraph ASSIGN["Vendor access"]
    assignment_req["POST /:id/vendor-assignments<br/>Tidak menyelesaikan node<br/>Perencanaan / Konstruksi<br/>Evidence: tidak"]
  end
  subgraph S4["Pra-konstruksi"]
    wotiang["POST /:id/wo-vendor/tiang<br/>wo_tiang · Perencanaan<br/>Evidence: wajib"]
    wokonstruksi["POST /:id/wo-vendor/konstruksi<br/>wo_konstruksi · Konstruksi<br/>Evidence + perlu_pdkb<br/>Jika perlu: tanggal_pdkb kosong + booking atomik"]
    reservasi["POST /:id/reservasi-material<br/>reservasi_material · Vendor Konstruksi<br/>Evidence: wajib"]
    tera["POST /:id/tera-app<br/>tera_app · Transaksi Energi<br/>Evidence: wajib"]
    wopdkb["POST /:id/wo-pdkb<br/>wo_pdkb · Konstruksi<br/>Evidence: wajib jika applicable"]
  end
  subgraph S5["Pelaksanaan"]
    installation["POST /:id/pelaksanaan-konstruksi<br/>workflow_node: pemasangan_tiang<br/>Vendor Tiang · Evidence: wajib"]
    construction["POST /:id/pelaksanaan-konstruksi<br/>workflow_node: pelaksanaan_konstruksi<br/>Vendor Konstruksi · Evidence: wajib"]
    pdkbdoc["POST /:id/pdkb-dokumentasi<br/>pdkb_documentation · PDKB<br/>Evidence: wajib jika applicable"]
  end
  subgraph S6["Penyalaan"]
    energize["POST /:id/energize-jaringan<br/>energize_jaringan<br/>Teknik · Jaringan<br/>Evidence + operation_result<br/>energize_status: temporary / permanent"]
    srapp["POST /:id/pemasangan-sr-app<br/>pemasangan_sr_app<br/>Transaksi Energi · semua jenis<br/>Evidence: wajib"]
  end
  subgraph S7["Penutupan"]
    closing["POST /:id/closing<br/>entri_mutasi_pdl + arsip_ail + selesai<br/>Pelayanan Pelanggan matching ULP<br/>Evidence: wajib · aggregate completed"]
  end

  create_req --> survey --> rab --> priority --> expansion
  expansion -. setelah delegated .-> assignment_req
  expansion -->|delegated dan perlu tiang| wotiang
  expansion --> wokonstruksi
  expansion --> tera
  wokonstruksi --> reservasi --> construction
  wokonstruksi --> wopdkb --> construction
  wokonstruksi -->|PDKB tidak diperlukan| construction
  wotiang --> installation --> energize
  construction -->|PDKB diperlukan| pdkbdoc
  construction -->|node konstruksi selesai| energize
  pdkbdoc --> energize
  tera --> srapp
  energize --> closing
  srapp --> closing

  survey_edit["PUT /:id/survei<br/>Role/unit pemilik survei<br/>Payload + evidence + expected_revision"]
  wo_edit["PUT /:id/wo-vendor/konstruksi/documents<br/>Konstruksi · lampiran upload + expected_revision"]
  revisions["GET /:id/activities/:workflow_node/revisions<br/>Riwayat survei / WO Konstruksi"]
  calendar["/api/pdkb/schedules<br/>GET: PDKB dan Konstruksi<br/>POST / PUT / DELETE: PDKB, kegiatan manual saja"]
  survey -. revisi setelah completed .-> survey_edit
  wokonstruksi -. revisi setelah completed .-> wo_edit
  survey_edit -. audit .-> revisions
  wo_edit -. audit .-> revisions
  calendar -. cek slot dan booking dalam transaksi .-> wokonstruksi

  class create_req,priority create
  class survey,rab,expansion,wotiang,wokonstruksi,reservasi,tera,wopdkb,installation,construction,pdkbdoc,energize,srapp,closing evidence
  class assignment_req assignment
```

Setiap activity submission mengembalikan aggregate terbaru dan caller-specific `available_actions`.
Wrong owner menghasilkan 403; node yang tidak actionable menghasilkan 409.

Perluasan dan delegasi NPS disubmit atomik. Hanya outcome `delegated` membuka
cabang berikutnya; `returned` menghentikan workflow. Panah masuk ke node workflow
menyatakan seluruh prerequisite, dengan skip sah untuk tiang/PDKB yang tidak perlu.
Diagram merangkum endpoint; dependency lengkap ada pada diagram per jenis.

PUT revisi tidak menyelesaikan ulang node atau mengubah SLA; tetap diizinkan
setelah closing/returned. Revision usang atau booking tanggal bentrok menghasilkan
409. Endpoint penyelesaian WO APP sudah dihapus dari routes aktif. Tanggal dan
nominal pembayaran wajib saat membuat permohonan baru.

Prefix `/api/permohonan` berlaku untuk semua path `/:id/...`; kalender memakai
prefix tersendiri seperti ditulis pada diagram. PUT/DELETE kalender menambahkan
`/:schedule_id`. GET riwayat tetap mengikuti read scope permohonan/activity.
