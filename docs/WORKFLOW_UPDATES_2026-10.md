# Workflow update — 24 October 2026

The agreed workbook/image revisions supplement the previous workflow diagrams.
Canonical node codes remain the transition and authorization identity; existing
activity numbers are retained as stable historical and SLA rule references.

## Active flow

1. Permohonan PB/PD captures payment date and amount.
2. Survey: Teknik of the matching ULP for JTR/JTM; Perencanaan for PLG TM.
3. RAB and the existing pole decision remain together.
4. **Verifikasi prioritas**: Perencanaan UP3 for every connection type, before
   Permohonan Perluasan/NPS. Confirmation plus optional notes; no new rejection.
5. NPS delegates or returns the request. Return remains terminal.
6. After delegation, pole WO (conditional), construction WO, and Tera APP run
   independently. WO APP is retired; historical completed nodes/files remain readable.
7. SR/APP and penyalaan are performed by **Transaksi Energi**, requiring Tera APP
   only. Construction, pole installation, material reservation, PDKB, and network
   energize keep their existing separate gates.
8. Network energize remains Teknik ULP for JTR/JTM and Jaringan for PLG TM. Its
   `energize_status` is `temporary` or `permanent`. Both can proceed to closing.
9. Closing still requires both network energize and SR/APP, and atomically completes
   PDL, AIL, and Selesai. Aggregate status remains in_progress/completed/returned.

## Revisions and evidence

- Survey date, notes, and uploaded attachments can be revised after completion,
  including closed/returned requests, by the original role/unit owner.
- Konstruksi UP3 can revise uploaded construction-WO attachments at any time after
  issuance. Generated WO PDF and other activity evidence associations are immutable.
- Revisions require `expected_revision`; stale edits return 409. Previous payload/file
  selections remain in history. Original completion actor/time and SLA are unchanged.
- Existing file validation remains 30 MiB per file and PDF/JPEG/PNG/XLS/XLSX.
- Landowner/village permissions remain optional survey categories, including a combined
  category. At least one survey evidence attachment remains required.

## PDKB calendar

- One team, one whole-day activity per civil date.
- PDKB manages manual activities; Konstruksi reads availability when issuing a WO.
- `tanggal_pdkb` is required when `perlu_pdkb=true` and is separate from the existing
  construction completion estimate.
- A unique date constraint reserves the slot in the same transaction as WO submission.
  Conflicts return 409 and roll back submission/evidence changes.
- Calendar WO bookings are locked against calendar edits/deletion. The UI marks occupied
  dates and disables them in the construction-WO date picker.
- PostgreSQL DATE values use UTC midnight with unchanged civil-date components; business
  calendar and today validation use WIB. This avoids a one-day pgx timezone shift.

## Payment-based working-day SLA

- New requests use `payment_workdays_v1`; existing requests/nodes keep `calendar` and
  their original stored deadlines. The migration never recalculates old deadlines.
- Working days: Monday–Friday in WIB, excluding national holidays. Collective leave
  remains a working day. A nonworking payment date shifts H/G to the first working day
  on or after payment; H+1 is the next working day.
- Existing offset rules stay unchanged, including distinct JTR/JTM/PLG TM deadlines.
  Priority uses the RAB deadline/offset.
- The backend returns remaining days, basis, status, and calendar-coverage errors.
  List filters use the same calendar through `colabora_sla_remaining` in PostgreSQL.
- Due soon means at most two working days. Reminders run on the preceding working day,
  overdue begins on the next working day, and weekends/holidays do not increase lateness.
- The static calendar covers 2026–2027. New calculations crossing coverage fail clearly;
  there is no silent calendar-day fallback and no external runtime holiday API.
- Official sources: [2026 SKB](https://www.kemenkopmk.go.id/node/5862),
  [2027 SKB](https://www.kemenkopmk.go.id/node/6444).

## Migration and delivery

- `20261024100000_workflow_updates`: revision/calendar tables, evidence-current marker,
  priority node, legacy-NPS exemption, historical WO APP retirement, energize substatus.
- Active requests before NPS receive the priority gate. Requests that already completed
  NPS receive a legitimate skipped priority node; progress and terminal history remain.
- `20261024110000_payment_workday_sla`: basis markers and the PostgreSQL SLA helper.
- Rollback refuses to discard populated workflow/revision data or new-basis requests.
- Deploy backend migrations and backend together, then the matching frontend.
- **Power catalogue expansion remains deferred**, as agreed; existing tariff/power data
  and dropdown behavior were not changed.

## Validation

Local validation uses a dedicated PostgreSQL database and Garage storage, synthetic
accounts/data, and no external email delivery. Tests cover 32 complete HTTP workflows
(four connection types × PB/PD × pole/no pole × PDKB/no PDKB), both energize outcomes,
NPS return, immutable completion/SLA on revisions, concurrent booking, real XLSX,
30-MiB boundary, migration compatibility, SQL/Go SLA parity, and date round trips.

Frontend: `node --test tests/api.test.cjs`, `npm run lint`, `npm run typecheck`,
`npm run build -- --webpack`. Backend: `go build ./...`, `go vet ./...`, and
`go test ./...` with CGO enabled; set `COLABORA_TEST_POSTGRES_DSN` for isolated
PostgreSQL migration tests. Windows without CGO can run the suite in `golang:1.26`.

Final local results: all Go packages (including PostgreSQL migration tests) passed;
Go build/vet, frontend lint/typecheck and the production Webpack build passed;
82 frontend tests passed. The existing `npm test` wildcard does not expand on the
local Windows/Node runtime, so the suite was run directly with
`node --test tests/api.test.cjs`. Live browser checks verified occupied PDKB dates,
calendar date persistence, payment fields, and revisions after closing.
Actual email delivery was not tested because local email credentials are absent.
