# SLA and Critical-Path View

**Status:** Implemented state, 9 September 2026  
**Perspektif:** Semua jenis sambungan

SLA adalah proyeksi dari request date dan bukan transition gate. Dependency graph
tetap menentukan node yang dapat dijalankan.

```mermaid
flowchart TB
  start["#1 Permohonan<br/>JTR/JTM H · PLG TM G"] --> planning["#2 Survei<br/>+2 hari"] --> rab["#3 RAB/KKO/KKF<br/>+2 hari"] --> expansion["#4 Perluasan<br/>JTR/JTM H+2 · PLG TM H"]
  expansion --> decision["#5 NPS delegation<br/>Tanpa SLA independen"]

  decision -->|delegated| parallel_start{"Cabang paralel"}
  decision -->|returned| returned(["Terminal returned"])
  parallel_start --> pole["#6 WO Tiang<br/>JTR/JTM H+2 · PLG TM H+1<br/>Conditional"]
  parallel_start --> construction_wo["#7 WO Konstruksi<br/>JTR/JTM H+2 · PLG TM H+1"]
  parallel_start --> app["#8 WO APP<br/>JTR/JTM H+2 · PLG TM H+1"]
  app --> reserve["#9 Reservasi<br/>H+2"] --> tera["#10 Tera APP<br/>JTR/JTM H+3 · PLG TM H+2"]
  pole --> install_pole["#11 Pemasangan Tiang<br/>H+6 / H+8 / H+8 / H+12"]
  construction_wo --> support["WO PDKB bila diperlukan<br/>Tanpa SLA independen"] --> construction["#12 Konstruksi<br/>H+8 / H+12 / H+19 / H+49"]
  construction --> pdkb["Dokumentasi PDKB<br/>Tanpa SLA independen"]
  install_pole --> energize["#13 Energize<br/>H+9 / H+13 / H+20 / H+50"]
  construction --> energize
  pdkb --> energize
  construction --> srapp["#14 SR/APP<br/>H+10 / H+14 / H+20 / H+50"]
  tera --> srapp
  energize --> join{"Stage 6 join"}
  srapp --> join
  join --> pdl["#15 PDL<br/>H+10 / H+10 / H+20 / H+50"] --> ail["#16 AIL/DIJ<br/>H+10 / H+14 / H+20 / H+50"] --> done["#17 Selesai<br/>H+10 / H+14 / H+20 / H+50"]

  classDef noSla fill:#f1f5f9,stroke:#64748b,color:#334155
  classDef join fill:#fef3c7,stroke:#b45309,color:#78350f
  class decision,support,pdkb noSla
  class parallel_start,join join
```

Urutan nilai yang dipisahkan `/` adalah JTR, JTM/Gardu, PLG TM &lt;5 GWNG, dan
PLG TM &gt;5 GWNG. Offset merupakan hari kalender dari request date. Runtime memakai
ambang `due soon` dua hari.
