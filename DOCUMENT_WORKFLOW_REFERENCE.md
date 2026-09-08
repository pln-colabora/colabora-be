# DOCUMENT_WORKFLOW_REFERENCE.md — Real-Document Findings

This document records domain findings from the real operational samples under
`BA & DOKUMEN COLABORA/`. It is intended to help design activity payloads,
document handling, and future production forms without coupling the backend to
the current hifi form pages.

The samples contain personal and operationally sensitive data. This document
therefore describes their structure without copying customer identifiers,
addresses, phone numbers, coordinates, signatures, financial values, or network
details.

## Authority and maintenance

- `hifi-colabora/workflow/jtr-jtm.html` and
  `hifi-colabora/workflow/plg-tm.html` remain the living source of truth for
  workflow order, dependencies, branches, and ownership.
- The files in `BA & DOKUMEN COLABORA/` are evidence and form-design references,
  not workflow definitions.
- The samples come from multiple customers and work packages. Their file dates
  must not be combined into one chronological lifecycle.
- Hifi forms and detail pages are disposable prototypes. Production forms may
  replace them while workflow-node identities and dependencies remain stable.
- Re-inspect this reference whenever newer operational samples are added. If a
  finding conflicts with the living diagrams, record the conflict and confirm it
  with the domain owner before changing workflow behavior.

## Sample coverage

The folder currently contains eleven business artefacts, excluding `.DS_Store`,
covering the following areas:

| Sample group | Likely workflow node(s) | What the evidence contains |
|---|---|---|
| Customer PB/PD application | `permohonan` (#1) | A customer-authored, signed application that may arrive as an unstructured scan or photo rather than a standard PLN form. |
| Survey drawings | `survei` (#2) | Customer and site context, network condition, technical recommendation, existing/planned topology, material needs, coordinates, and field approvals. |
| RAB workbook | `rab_kko_kkf` (#3) and `kebutuhan_tiang` (#3b) | Customer and technical inputs, construction catalogues, prices, transformer/load references, formulas, feasibility analysis, and printable outputs. |
| NPS work instruction | `permohonan_perluasan` (#4) and `nps_delegation` (#5) | Customer, commercial, payment, technical, location, equipment, service, and authorization data in a formal Perintah Kerja. |
| Pole-vendor WO | `wo_tiang` (#6) | Work volume, pricing, contract reference, vendor SLA, supervisor/contact data, technical drawing, and authorization. |
| Construction-vendor package | `wo_konstruksi` (#7), with content related to `reservasi_material` (#9) | A multi-page package containing drawings, work references, customer appendices, and multiple material-reservation slips. |
| Relay/OCR setting sheet | Most likely `tera_app` (#10) | Relay identity, voltage, PT/CT ratio, old/new power, calculated current, protection settings, staged trip tests, conclusion, and sign-off. |
| Network operation BA | `energize_jaringan` (#13) | Work-instruction reference, gardu/feeder, old/new power, cubicle/equipment identity, insulation or resistance measurements, operation result, and sign-off. |

The current samples do not provide complete evidence for `wo_app` (#8),
`pemasangan_tiang` (#11), `pelaksanaan_konstruksi` (#12),
`pemasangan_sr_app` (#14), activities #15–#17, or the PDKB branch. Do not invent
their detailed payloads from the existing prototype alone.

## Workflow conclusions supported by the samples

1. Customer application begins the case, but its evidence can be unstructured.
   The application record and the uploaded source file should be treated as
   separate concerns.
2. Survey output feeds planning. The drawings and RAB use customer, location,
   network, load, and material information that must already be known before NPS
   can issue a work instruction.
3. The NPS step is a delegation decision. The observed output is a formal
   **Perintah Kerja**, supporting the target state names `delegated` and
   `returned` rather than a generic approval status.
4. After delegation, pole, construction, and APP preparation are independent or
   conditional branches as defined by the living workflow. Activity numbers must
   not be used as a simple incrementing state machine.
5. Relay setting and functional testing are technical preparation/commissioning
   evidence, not merely generic attachments. Confirm the final node boundary with
   Transaksi Energi when the production form is specified.
6. Network operation is a measured technical event. Completing it should capture
   the structured result and the authorized BA, not only an upload flag.

## Form and payload design implications

### Separate source evidence, structured input, and generated output

For each workflow node, distinguish these concepts:

- **source evidence** — customer letters, external drawings, photos, or existing
  signed documents uploaded without attempting to reconstruct their content;
- **structured activity payload** — validated fields needed by workflow rules,
  searching, reporting, calculations, or downstream nodes;
- **generated artefact** — a versioned PDF/XLSX produced from structured data for
  review, printing, signing, or archiving.

Do not place all data in document metadata, and do not treat the existence of an
uploaded file as proof that required structured fields are valid.

### Keep workflow-critical fields typed

Values that determine applicability, ownership, authorization, dependencies, or
aggregate status belong in typed columns or validated DTO fields. Examples
include connection type, target ULP, pole requirement, NPS delegation result,
PDKB requirement, and completion outcome.

Node-specific measurements and form values may remain in validated JSONB payloads
until the detailed forms stabilize. Promote fields to typed columns when they are
queried across requests, reused downstream, or participate in business rules.

### Treat the RAB workbook as a calculation system

The inspected workbook has 33 worksheets and includes large formula-driven
catalogues and analyses. Its concerns include:

- customer and request input;
- technical drawing input;
- material and construction catalogues;
- price/reference tables;
- transformer and measurement data;
- regular, aesthetic, and premium feasibility analyses;
- projections and financial calculations;
- printable recap/output sheets.

Replacing this workbook with one large web form would embed unstable calculations
in the UI. A future implementation should instead separate:

1. validated inputs;
2. versioned reference/catalogue data;
3. a testable calculation service;
4. persisted calculation results and the ruleset version used;
5. review/approval state;
6. generated immutable output documents.

Until formulas and reference ownership have been validated with the business,
retain the original workbook as evidence and do not claim calculation parity.

### Model repeatable technical sections

Several documents contain repeatable rows rather than fixed scalar fields:

- material reservation lines;
- existing and planned construction components;
- equipment identities and serial numbers;
- electrical measurements by phase/pair;
- relay settings and staged test results;
- authorization/signature roles.

Production payloads should use arrays of typed row objects where practical. Avoid
fields such as `material_1`, `material_2`, or a single unrestricted notes string.

### Preserve document provenance

Document records will likely need more than type and object path. Candidate
metadata to validate during the persistence refactor:

- document type and template/version;
- external document/work-order number;
- revision number;
- issuer and issuer unit;
- issued, effective, signed, and uploaded timestamps where applicable;
- original filename, MIME type, byte size, checksum, and storage object key;
- source (`uploaded`, `generated`, `imported`, or `scanned`);
- confidentiality/classification;
- superseded/replaced relationship;
- signatory roles or signature status, without placing signature images in logs.

## Document-to-workflow association

The current target model places one nullable `WorkflowNode` directly on each
document. The samples show that a single physical file can be a bundle containing
content relevant to multiple nodes, especially a construction WO package that
also includes material-reservation slips.

Before finalizing document persistence, make one explicit product decision:

1. **Single-node artefacts:** split bundles during ingestion and require every
   stored document to be classified under exactly one workflow node; or
2. **Reusable source files:** store the physical file once and associate it with
   one or more workflow-node evidence records through a separate attachment or
   classification table.

Do not preserve the statement “a document is never attached to more than one
activity, ever” without validating this choice against the observed bundles.
Whichever model is selected, node completion must still validate the evidence
requirements for that node explicitly.

## Security and repository handling

The sample directory contains personally identifiable information and sensitive
operational data, including customer identifiers, exact addresses, phone
numbers, coordinates, signatures, pricing, equipment, and network information.

- Keep original samples out of public Git history unless they have been formally
  approved and sanitized.
- `.DS_Store` is already ignored by the root `.gitignore`, but the rest of
  `BA & DOKUMEN COLABORA/` is not ignored automatically.
- Before staging broad changes, check `git status --short` and stage documentation
  or code with explicit paths.
- Never copy real values into seeders, automated tests, screenshots, API examples,
  logs, issue descriptions, or generated documentation.
- If fixtures are needed, create synthetic documents with clearly fictional data.
- Production uploads must remain private and require authorized, short-lived
  download access. Validate MIME and size, scan untrusted files, retain audit
  events, and close the known evidence-view authorization gap before rollout.

## Questions for detailed-form discovery

Resolve these with each business owner before implementing production forms:

- Which fields are entered in COLABORA, imported from another PLN system, or only
  displayed on a generated document?
- Which fields are mandatory for node completion, and which are optional evidence?
- Who authors, reviews, signs, revises, and can supersede each document?
- Does a correction reopen a completed node or create a new revision?
- Which calculations and price/catalogue versions are authoritative for RAB?
- Is relay/OCR testing wholly part of `tera_app`, or does it gate another node?
- Must bundled WO/reservation files be split, or may one source file evidence
  multiple workflow nodes?
- What exact evidence is required for the currently uncovered nodes and PDKB
  branch?
- Which values must flow forward automatically to later nodes, and which may be
  corrected independently?

## Developer checklist

Before implementing or changing a workflow form:

1. Re-read both files under `hifi-colabora/workflow/`.
2. Identify the stable workflow-node code, connection type, owner, prerequisites,
   applicability conditions, and join gate.
3. Compare the relevant real sample with the proposed DTO without copying PII.
4. Classify each field as workflow-critical, reusable business data,
   node-specific payload, calculated output, or document-only content.
5. Define required evidence and document provenance metadata.
6. Confirm revision/correction behavior and transaction boundaries.
7. Add service-level validation and dependency tests before exposing the handler.
8. Update `PRD.md`, `DATA_MODEL.md`, `API_SPEC.md`, `RBAC.md`, and runtime OpenAPI
   only where the implementation or agreed target contract has actually changed.

