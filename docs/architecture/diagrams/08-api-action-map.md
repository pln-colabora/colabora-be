# API Action Map

**Status:** Implemented state, 9 September 2026  
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
    expansion["POST /:id/permohonan-perluasan<br/>permohonan_perluasan + nps_delegation<br/>NPS<br/>Evidence: wajib"]
  end
  subgraph ASSIGN["Vendor access"]
    assignment_req["POST /:id/vendor-assignments<br/>Tidak menyelesaikan node<br/>Perencanaan / Konstruksi / Transaksi Energi<br/>Evidence: tidak"]
  end
  subgraph S4["Pra-konstruksi"]
    wotiang["POST /:id/wo-vendor/tiang<br/>wo_tiang · Perencanaan<br/>Evidence: wajib"]
    wokonstruksi["POST /:id/wo-vendor/konstruksi<br/>wo_konstruksi · Konstruksi<br/>Evidence + perlu_pdkb"]
    woapp["POST /:id/wo-vendor/app<br/>wo_app · Transaksi Energi<br/>Evidence: wajib"]
    reservasi["POST /:id/reservasi-material<br/>reservasi_material + tera_app<br/>Transaksi Energi · Evidence: wajib"]
    wopdkb["POST /:id/wo-pdkb<br/>wo_pdkb · Konstruksi<br/>Evidence: wajib jika applicable"]
    pkvendor["POST /:id/pk-vendor<br/>pk_vendor · Konstruksi<br/>Evidence: wajib"]
  end
  subgraph S5["Pelaksanaan"]
    construction["POST /:id/pelaksanaan-konstruksi<br/>pemasangan_tiang atau pelaksanaan_konstruksi<br/>Matching vendor · Evidence: wajib"]
    pdkbdoc["POST /:id/pdkb-dokumentasi<br/>pdkb_documentation · PDKB<br/>Evidence: wajib jika applicable"]
  end
  subgraph S6["Penyalaan"]
    energize["POST /:id/energize-jaringan<br/>energize_jaringan<br/>Teknik · Jaringan<br/>Evidence + operation_result"]
    srapp["POST /:id/pemasangan-sr-app<br/>pemasangan_sr_app<br/>Vendor SR/APP · Vendor Konstruksi<br/>Evidence: wajib"]
  end
  subgraph S7["Penutupan"]
    closing["POST /:id/closing<br/>entri_mutasi_pdl + arsip_ail + selesai<br/>Pelayanan Pelanggan matching ULP<br/>Evidence: wajib · aggregate completed"]
  end

  create_req --> survey --> rab --> expansion
  expansion -. setelah delegated .-> assignment_req
  expansion --> wotiang
  expansion --> wokonstruksi
  expansion --> woapp --> reservasi
  wokonstruksi --> wopdkb --> pkvendor
  wokonstruksi -->|PDKB tidak diperlukan| pkvendor
  wotiang --> construction
  pkvendor --> construction
  construction --> pdkbdoc
  construction --> energize
  pdkbdoc --> energize
  construction --> srapp
  reservasi --> srapp
  energize --> closing
  srapp --> closing

  class create_req create
  class survey,rab,expansion,wotiang,wokonstruksi,woapp,reservasi,wopdkb,pkvendor,construction,pdkbdoc,energize,srapp,closing evidence
  class assignment_req assignment
```

Path pada diagram menggunakan prefix bersama `/api/permohonan`. Setiap activity
submission mengembalikan aggregate terbaru dan caller-specific `available_actions`.
Wrong owner menghasilkan 403; node yang tidak actionable menghasilkan 409.
