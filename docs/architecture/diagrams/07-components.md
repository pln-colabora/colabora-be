# Component Diagram

**Status:** Implemented state, 9 September 2026  
**Perspektif:** Runtime backend dan layanan eksternal

```mermaid
flowchart LR
  client["Frontend / API Client"]

  subgraph APP["COLABORA Go Backend"]
    gin["Gin Engine<br/>cmd/main.go"]
    cors["CORS"]
    authmw["JWT Authenticate"]
    access["Permohonan Access Guard"]

    subgraph MODULES["Feature Modules"]
      auth["Auth<br/>Controller → Service → Repository"]
      user["User<br/>Controller → Service → Repository"]
      permohonan["Permohonan<br/>Controller → Service → Repository"]
      document["Document<br/>Controller → Service → Repository"]
    end

    workflow["pkg/workflow<br/>Pure dependency evaluator"]
    rbac["pkg/rbac<br/>Node ownership + read scope"]
    di["samber/do<br/>providers/core.go"]
    migrations["Migrations + Seeders"]
    docs["Scalar + Architecture Docs"]
  end

  postgres[("PostgreSQL")]
  garage[("Garage / S3-compatible<br/>private objects")]
  clamav["ClamAV<br/>optional synchronous scanner"]
  email["Email transport"]

  client --> gin
  gin --> cors
  gin --> authmw --> access
  gin --> auth
  gin --> user
  access --> permohonan
  access --> document
  gin --> docs
  di -. constructs .-> auth
  di -. constructs .-> user
  di -. constructs .-> permohonan
  di -. constructs .-> document
  permohonan --> workflow
  permohonan --> rbac
  document --> rbac
  auth --> postgres
  user --> postgres
  permohonan --> postgres
  document --> postgres
  migrations --> postgres
  document --> clamav
  document --> garage
  auth --> email

  classDef domain fill:#fef3c7,stroke:#b45309,color:#78350f
  classDef infra fill:#e2e8f0,stroke:#475569,color:#0f172a
  class workflow,rbac domain
  class postgres,garage,clamav,email infra
```

Repository menerima `*gorm.DB` atau transaksi dari service. Evaluator workflow tidak
melakukan I/O; service memegang transaction boundary dan orchestration.
