# SLA and Critical-Path View

**Status:** Implementasi branch dev, 26 Oktober 2026
**Perspektif:** Semua jenis sambungan

SLA permohonan baru memakai tanggal pembayaran dan basis `payment_workdays_v1`.
Dependency graph tetap menentukan node yang dapat dijalankan; deadline tidak
menjadi transition gate. Semua panah masuk adalah prerequisite bersama, dengan
skip sah untuk cabang tiang/PDKB yang tidak applicable.

```mermaid
flowchart TB
  payment["Tanggal pembayaran pada permohonan"] --> normalize["H/G = hari kerja pertama pada / setelah pembayaran<br/>WIB · Senin–Jumat kecuali libur nasional"]
  normalize --> start["#1 Permohonan<br/>JTR/JTM H · PLG TM G"] --> planning["#2 Survei<br/>+2 hari"] --> rab["#3 RAB/KKO/KKF<br/>+2 hari"] --> priority["Verifikasi prioritas<br/>Deadline mengikuti RAB: +2 hari kerja"] --> expansion["#4 Perluasan<br/>JTR/JTM H+2 · PLG TM H"]
  expansion --> decision["#5 NPS delegation<br/>Tanpa SLA independen"]

  decision -->|delegated| parallel_start{"Cabang paralel"}
  decision -->|returned| returned(["Terminal returned"])
  parallel_start --> pole["#6 WO Tiang<br/>JTR/JTM H+2 · PLG TM H+1<br/>Conditional"]
  parallel_start --> construction_wo["#7 WO Konstruksi<br/>JTR/JTM H+2 · PLG TM H+1"]
  parallel_start --> tera["#10 Tera APP<br/>JTR/JTM H+3 · PLG TM H+2"]
  pole --> install_pole["#11 Pemasangan Tiang<br/>H+6 / H+8 / H+8 / H+12"]
  construction_wo --> reserve["#9 Reservasi<br/>H+2"] --> construction["#12 Konstruksi<br/>H+8 / H+12 / H+19 / H+49"]
  construction_wo --> support["WO PDKB bila diperlukan<br/>Tanpa SLA independen"] --> construction
  construction --> pdkb["Dokumentasi PDKB<br/>Tanpa SLA independen"]
  install_pole --> energize["#13 Energize · sementara / tetap<br/>H+9 / H+13 / H+20 / H+50"]
  construction --> energize
  pdkb --> energize
  tera --> srapp["#14 SR/APP · Transaksi Energi<br/>H+10 / H+14 / H+20 / H+50"]
  energize --> join{"Stage 6 join"}
  srapp --> join
  join --> pdl["#15 PDL<br/>H+10 / H+10 / H+20 / H+50"] --> ail["#16 AIL/DIJ<br/>H+10 / H+14 / H+20 / H+50"] --> done["#17 Selesai<br/>H+10 / H+14 / H+20 / H+50"]

  classDef noSla fill:#f1f5f9,stroke:#64748b,color:#334155
  classDef join fill:#fef3c7,stroke:#b45309,color:#78350f
  class decision,support,pdkb noSla
  class parallel_start,join join
```

Urutan nilai yang dipisahkan `/` adalah JTR, JTM/Gardu, PLG TM &lt;5 GWNG, dan
PLG TM &gt;5 GWNG. Offset di atas merupakan hari kerja dari H/G untuk permohonan baru, bukan durasi
setelah kegiatan sebelumnya. H+1 berarti hari kerja berikutnya. Cuti bersama tetap
dihitung jika jatuh pada hari kerja dan bukan libur nasional.

Basis `calendar` pada permohonan lama mempertahankan deadline hari kalender dari
`request_date`; migrasi tidak menghitung ulang deadline historis. Verifikasi
prioritas menggunakan rule #3; keputusan tiang, delegasi NPS, WO PDKB, dan
dokumentasi PDKB tidak memiliki deadline independen.

Sisa hari/status dari backend memakai kalender yang sama dengan filter list dan
reminder. `due_soon` sampai dua hari kerja sebelum deadline, `overdue` mulai hari
kerja berikutnya; akhir pekan/libur tidak menambah keterlambatan. Node completed
menghitung sisa hari terhadap waktu completion, bukan waktu revisi.

Kalender statis mencakup 2026–2027. Perhitungan di luar cakupan mengembalikan error
kalender tanpa fallback hari kalender. Tidak ada panggilan API libur saat runtime.

Sumber: `pkg/businessday`, `database/entities/workflow.go`, dan
`database/seeders/json/sla_rules.json`. PDL JTM/Gardu tetap H+10 sesuai seed;
AIL/Selesai JTM/Gardu H+14. Closing ketiganya tetap dilakukan atomik.
