# Workflow PLG TM

**Status:** Implemented state, 9 September 2026  
**Perspektif:** PLG TM &lt;5 GWNG dan PLG TM &gt;5 GWNG

Dependency sama untuk kedua varian PLG TM; SLA-nya berbeda. Ownership awal, survei,
energize, dan SR/APP berbeda dari JTR/JTM.

```mermaid
flowchart TD
  classDef ulp fill:#dbeafe,stroke:#1d4ed8,color:#1e3a8a
  classDef up3 fill:#ede9fe,stroke:#6d28d9,color:#4c1d95
  classDef nps fill:#fef3c7,stroke:#b45309,color:#78350f
  classDef vendor fill:#ccfbf1,stroke:#0f766e,color:#134e4a
  classDef pdkb fill:#fce7f3,stroke:#be185d,color:#831843
  classDef decision fill:#f1f5f9,stroke:#475569,color:#0f172a
  classDef terminal fill:#fee2e2,stroke:#b91c1c,color:#7f1d1d

  subgraph S1["Stage 1 · Permohonan"]
    permohonan["#1 permohonan<br/>NPS · pilih ULP tujuan"]
  end
  subgraph S2["Stage 2 · Survei"]
    survei["#2 survei<br/>Perencanaan · UP3"]
  end
  subgraph S3["Stage 3 · Perencanaan"]
    rab_kko_kkf["#3 rab_kko_kkf<br/>Perencanaan · UP3"]
    kebutuhan_tiang{"kebutuhan_tiang<br/>Tiang diperlukan?"}
    permohonan_perluasan["#4 permohonan_perluasan<br/>NPS · UP3"]
    nps_delegation{"#5 nps_delegation<br/>NPS · UP3"}
  end
  subgraph S4["Stage 4 · Pra-konstruksi paralel"]
    wo_tiang["#6 wo_tiang<br/>Perencanaan"]
    wo_konstruksi["#7 wo_konstruksi<br/>Konstruksi"]
    wo_pdkb["wo_pdkb<br/>Konstruksi"]
    pk_vendor["pk_vendor<br/>Konstruksi"]
    wo_app["#8 wo_app<br/>Transaksi Energi"]
    reservasi_material["#9 reservasi_material<br/>Transaksi Energi"]
    tera_app["#10 tera_app<br/>Transaksi Energi"]
  end
  subgraph S5["Stage 5 · Pelaksanaan paralel"]
    pemasangan_tiang["#11 pemasangan_tiang<br/>Vendor Tiang"]
    pelaksanaan_konstruksi["#12 pelaksanaan_konstruksi<br/>Vendor Konstruksi"]
    pdkb_documentation["pdkb_documentation<br/>PDKB"]
  end
  subgraph S6["Stage 6 · Penyalaan paralel"]
    energize_jaringan["#13 energize_jaringan<br/>Jaringan · UP3"]
    pemasangan_sr_app["#14 pemasangan_sr_app<br/>Vendor Konstruksi"]
  end
  subgraph S7["Stage 7 · Closing atomik"]
    entri_mutasi_pdl["#15 entri_mutasi_pdl<br/>Pelayanan Pelanggan · ULP"]
    arsip_ail["#16 arsip_ail<br/>Pelayanan Pelanggan · ULP"]
    selesai["#17 selesai<br/>Pelayanan Pelanggan · ULP"]
  end
  returned(["returned · terminal"])

  permohonan --> survei --> rab_kko_kkf --> kebutuhan_tiang --> permohonan_perluasan --> nps_delegation
  nps_delegation -->|returned| returned
  nps_delegation -->|delegated + tiang=true| wo_tiang
  nps_delegation -->|delegated| wo_konstruksi
  nps_delegation -->|delegated| wo_app
  wo_tiang --> pemasangan_tiang
  wo_konstruksi -->|perlu_pdkb=true| wo_pdkb --> pk_vendor
  wo_konstruksi -->|perlu_pdkb=false; wo_pdkb skipped| pk_vendor
  pk_vendor --> pelaksanaan_konstruksi
  wo_pdkb --> pelaksanaan_konstruksi
  wo_app --> reservasi_material --> tera_app
  pemasangan_tiang --> energize_jaringan
  pelaksanaan_konstruksi -->|perlu_pdkb=true| pdkb_documentation
  pelaksanaan_konstruksi -->|perlu_pdkb=false; documentation skipped| energize_jaringan
  pdkb_documentation --> energize_jaringan
  pelaksanaan_konstruksi --> pemasangan_sr_app
  tera_app --> pemasangan_sr_app
  energize_jaringan --> entri_mutasi_pdl
  pemasangan_sr_app --> entri_mutasi_pdl
  entri_mutasi_pdl --> arsip_ail --> selesai

  class energize_jaringan,entri_mutasi_pdl,arsip_ail,selesai ulp
  class survei,rab_kko_kkf,wo_tiang,wo_konstruksi,wo_pdkb,pk_vendor,wo_app,reservasi_material,tera_app up3
  class permohonan,permohonan_perluasan,nps_delegation nps
  class pemasangan_tiang,pelaksanaan_konstruksi,pemasangan_sr_app vendor
  class pdkb_documentation pdkb
  class kebutuhan_tiang decision
  class returned terminal
```

Legenda warna: biru = ULP, ungu = fungsi UP3, kuning = NPS, hijau = vendor,
merah muda = PDKB, abu-abu = keputusan, merah = terminal.
