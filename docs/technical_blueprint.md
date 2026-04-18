# Dev Invoice App — Technical Blueprint (Pragmatic Baseline)

## 1. Scope

This system is a Go application for managing:

* legal entities shared by issuer and customer roles
* customer profiles
* customer-profile-specific service agreements
* billable time entries
* invoice generation
* invoice PDF rendering
* CLI execution
* MCP tool execution
* request-authenticated protected access

This baseline stays intentionally pragmatic.

It does **not** include:

* tax engine
* multi-user auth
* multi-tenant support
* CQRS
* event sourcing
* message broker
* workflow engine
* distributed services

The main objective is to support a reliable hourly billing flow with a clean internal model, simple delivery mechanisms, and direct compatibility with CLI and MCP connectors.

---

## 2. Architecture Direction

### Chosen style

* Pragmatic modular monolith
* Simple layered architecture
* Clear internal boundaries
* Connector-friendly design for CLI and MCP

### What this means

* We keep the billing model explicit.
* We avoid forcing heavy DDD structure everywhere.
* We use small business models and services where they help.
* We expose operations through application commands/services.
* We keep persistence, PDF rendering, auth, CLI, and MCP outside the core logic.

### What we are explicitly avoiding

* aggregate obsession
* repository-per-concept dogma
* complex port taxonomy where it adds no value
* over-modeled domain language for a small product
* premature infrastructure complexity

---

## 3. Frozen Decisions

### Persistence

* SQLite only
* No encryption in the initial version
* No Turso
* No cloud sync in the initial version

### Access model

* Single local operator
* API key authentication for MCP HTTP (Bearer token, validated in middleware)
* MCP HTTP protected operations require bearer-authenticated request identity
* CLI uses a local identity constructed at startup; no allowlist enforcement

### Billing model

* Hourly billing first
* Customer profile can have one or more service agreements
* One service agreement has one hourly rate at a time
* Invoice quantity is total billable hours

### Currency

* USD default
* Base support for future multi-currency
* No exchange-rate logic yet
* No FX conversion yet

### Precision

* Money stored as integer
* Hours stored as integer
* Money precision: 4 digits
* Hours precision: 4 digits
* Invoice display preserves 4-digit precision where required

### Tax policy

* No tax calculation in the initial version
* Structure should allow future extension

### PDF rendering

* Prefer the approach that most easily reproduces the current invoice layout
* HTML-to-PDF is preferred over low-level drawing unless proven insufficient
* PDF generation stays behind a rendering service boundary

---

## 4. Design Principles

1. Keep business rules in plain Go types and services.
2. Keep the architecture easy to navigate.
3. CLI and MCP should call the same application services.
4. Persistence should not leak into billing logic.
5. Authentication should not leak into billing logic.
6. PDF rendering should not own invoice calculations.
7. Issued invoices must remain reproducible.
8. Prefer boring code over theoretical purity.
9. Start with the smallest structure that supports the whole billing flow.

---

## 5. System Shape

Use four practical layers.

## 5.1 Core

Contains the billing model and core rules.

Includes:

* legal entity
* customer profile
* service agreement
* time entry
* invoice
* issuer profile
* money
* hours
* currency
* business validation rules

## 5.2 Application

Contains the operations the app exposes.

Includes:

* create legal entity
* create customer profile
* configure issuer profile
* create agreement
* record time entry
* create invoice draft
* issue invoice
* render invoice pdf
* inspect current authenticated identity state
* validate authenticated access

## 5.3 Connectors

Contains entrypoints.

Includes:

* CLI commands
* MCP tools
* auth callback handlers if needed

These adapters translate input/output and call application services.

## 5.4 Infrastructure

Contains technical implementation details.

Includes:

* SQLite access
* repositories/store layer
* API key auth (MCP HTTP)
* PDF renderer
* config loading

---

## 6. Core Data Model

The app only needs a few stable business objects.

## 6.1 LegalEntity

Represents shared legal and contact data for a billable party.

### Fields

* `ID`
* `Type` (`individual`, `company`)
* `LegalName`
* `TradeName`
* `TaxID`
* `Email`
* `Phone`
* `Website`
* `BillingAddress`
* `CreatedAt`
* `UpdatedAt`

### Rules

* a legal entity must have a billing name
* type must be one of `individual` or `company`

---

## 6.2 CustomerProfile

Represents a customer role attached 1:1 to a legal entity.

### Fields

* `ID`
* `LegalEntityID`
* `Status`
* `DefaultCurrency`
* `Notes`
* `CreatedAt`
* `UpdatedAt`

### Rules

* belongs to exactly one legal entity
* one legal entity can have at most one customer profile
* inactive customer profiles should not receive new invoices
* default currency is USD unless explicitly set otherwise

---

## 6.3 ServiceAgreement

Represents a billable agreement for a customer profile.

### Fields

* `ID`
* `CustomerProfileID`
* `Name`
* `Description`
* `BillingMode`
* `HourlyRate`
* `Currency`
* `Active`
* `ValidFrom`
* `ValidUntil`
* `CreatedAt`
* `UpdatedAt`

### Rules

* belongs to exactly one customer profile
* initial version supports `hourly` only
* hourly rate must be positive
* agreement must be active to record billable time
* rate changes must not mutate issued invoices

---

## 6.4 TimeEntry

Represents recorded work.

### Fields

* `ID`
* `CustomerProfileID`
* `ServiceAgreementID`
* `WorkDate`
* `Hours`
* `Description`
* `Billable`
* `InvoiceID` (optional)
* `CreatedAt`
* `UpdatedAt`

### Rules

* hours must be greater than zero
* billable time must reference an active agreement
* one time entry can belong to one invoice only
* once invoiced, it becomes financially logouted
* non-billable entries must not be included in invoice generation

---

## 6.5 Invoice

Represents a financial document.

### Fields

* `ID`
* `InvoiceNumber`
* `CustomerProfileID`
* `IssueDate`
* `DueDate`
* `Currency`
* `Status` (`draft`, `issued`, `discarded`)
* `Notes`
* `Subtotal`
* `GrandTotal`
* `Lines`
* `IssuedAt`
* `DiscardedAt`
* `CreatedAt`
* `UpdatedAt`

### InvoiceLine fields

* `ID`
* `ServiceAgreementID` (optional)
* `Description`
* `Quantity`
* `UnitPrice`
* `LineSubtotal`
* `LineTotal`
* `SourceType`
* `SourceRef`
* `SortOrder`

### Rules

* an invoice must have at least one line
* totals are always derived from lines
* currency must be internally consistent
* draft invoices are editable
* issued invoices are immutable
* grand total equals subtotal in the first version
* issued invoices preserve rate snapshots
* discard on draft: hard-delete invoice + lines, unlock time entries (atomic transaction)
* discard on issued: soft-delete — status transitions to "discarded", number permanently consumed, time entries remain locked
* discarded invoices reject further discard attempts
* invoice numbers are never reused even after soft-discard

---

## 6.6 IssuerProfile

Represents the operator/company issuing invoices.

### Fields

* `ID`
* `LegalEntityID`
* `DefaultCurrency`
* `DefaultNotes`
* `CreatedAt`
* `UpdatedAt`

### Rules

* belongs to exactly one legal entity
* one legal entity can have at most one issuer profile

---

## 6.7 Session

Represents current access state.

### Fields

* `ID`
* `UserEmail`
* `Unlogouted`
* `UnlogoutedAt`
* `LastActivityAt`
* `LockedAt`

### Rules

* no protected operation before authenticated identity is present
* MCP HTTP requires valid bearer-authenticated API key identity on each request
* CLI constructs a local identity at startup; protected operations receive it automatically

---

## 7. Shared Types

## 7.1 Money

### Rules

* stored as integer
* precision fixed to 4 digits
* always carries currency
* no float usage

### Suggested shape

* `Amount int64`
* `Currency string`

Example:

* `156250` => `15.6250`
* `25000000` => `2500.0000`

### Operations

* add
* subtract
* multiply by quantity
* compare
* format for invoice display

---

## 7.2 Hours

### Rules

* stored as integer
* precision fixed to 4 digits
* no float usage

### Suggested shape

* `Value int64`

Example:

* `1600000` => `160.0000`
* `12500` => `1.2500`

### Operations

* add
* compare
* validate positive
* format for invoice display

---

## 7.3 Other shared types

* `CurrencyCode`
* `InvoiceNumber`
* `Address`
* `DateRange`
* `CustomerProfileStatus`
* `InvoiceStatus`
* `BillingMode`

---

## 8. Application Services

Keep the app surface small and explicit.

## 8.1 Access

* `GetSessionStatus`
* `ValidateAccess`

## 8.2 Legal Entity

* `CreateLegalEntity`
* `UpdateLegalEntity`
* `GetLegalEntity`
* `ListLegalEntities`
* `DeleteLegalEntity`

## 8.3 Customer Profile

* `CreateCustomerProfile`
* `UpdateCustomerProfile`
* `GetCustomerProfile`
* `ListCustomerProfiles`
* `DeleteCustomerProfile`

## 8.4 Service Agreement

* `CreateServiceAgreement`
* `UpdateServiceAgreementRate`
* `ActivateServiceAgreement`
* `DeactivateServiceAgreement`
* `ListCustomerProfileServiceAgreements`

## 8.5 Time Entry

* `RecordTimeEntry`
* `UpdateTimeEntry`
* `DeleteDraftTimeEntry`
* `ListCustomerProfileTimeEntries`
* `ListUnbilledTimeEntries`

## 8.6 Invoice

* `CreateDraftInvoice`
* `CreateDraftInvoiceFromUnbilledTime`
* `AddManualInvoiceLine`
* `RemoveDraftInvoiceLine`
* `RecalculateInvoiceTotals`
* `IssueInvoice`
* `DiscardInvoice` — mixed semantics: draft invoices are hard-deleted (entries unlocked); issued invoices are soft-discarded (status → discarded, number permanently consumed, entries remain locked)
* `GetInvoice`
* `ListInvoices`
* `RenderInvoicePDF`

## 8.7 Issuer Profile

* `CreateIssuerProfile`
* `UpdateIssuerProfile`
* `GetIssuerProfile`

---

## 9. Connector Interfaces

These are enough. No need to overname them as formal ports everywhere.

```go
type LegalEntityService interface {
    Create(ctx context.Context, cmd CreateLegalEntityCommand) (LegalEntityDTO, error)
    Update(ctx context.Context, id string, cmd PatchLegalEntityCommand) (LegalEntityDTO, error)
    Get(ctx context.Context, id string) (LegalEntityDTO, error)
    List(ctx context.Context, query ListQuery) (ListResult[LegalEntityDTO], error)
    Delete(ctx context.Context, id string) error
}

type CustomerProfileService interface {
    Create(ctx context.Context, cmd CreateCustomerProfileCommand) (CustomerProfileDTO, error)
    Update(ctx context.Context, id string, cmd PatchCustomerProfileCommand) (CustomerProfileDTO, error)
    Get(ctx context.Context, id string) (CustomerProfileDTO, error)
    List(ctx context.Context, query ListQuery) (ListResult[CustomerProfileDTO], error)
    Delete(ctx context.Context, id string) error
}

type IssuerProfileService interface {
    Create(ctx context.Context, cmd CreateIssuerProfileCommand) (IssuerProfileDTO, error)
    Update(ctx context.Context, id string, cmd PatchIssuerProfileCommand) (IssuerProfileDTO, error)
    Get(ctx context.Context, id string) (IssuerProfileDTO, error)
}

type AgreementService interface {
    Create(ctx context.Context, cmd CreateServiceAgreementCommand) (ServiceAgreementDTO, error)
    Get(ctx context.Context, id string) (ServiceAgreementDTO, error)
    UpdateRate(ctx context.Context, id string, cmd UpdateServiceAgreementRateCommand) (ServiceAgreementDTO, error)
    Activate(ctx context.Context, id string) (ServiceAgreementDTO, error)
    Deactivate(ctx context.Context, id string) (ServiceAgreementDTO, error)
    ListByCustomerProfile(ctx context.Context, customerProfileID string) ([]ServiceAgreementDTO, error)
}

type TimeEntryService interface {
    Record(ctx context.Context, cmd RecordTimeEntryCommand) (TimeEntryDTO, error)
    ListUnbilled(ctx context.Context, customerProfileID string, period DateRangeDTO) ([]TimeEntryDTO, error)
}

type InvoiceService interface {
    CreateDraftFromUnbilled(ctx context.Context, cmd CreateDraftFromUnbilledCommand) (InvoiceDTO, error)
    IssueDraft(ctx context.Context, cmd IssueInvoiceCommand) (InvoiceDTO, error)
    Discard(ctx context.Context, invoiceID string) (DiscardResult, error)
    GetInvoice(ctx context.Context, invoiceID string) (InvoiceDTO, error)
    ListInvoices(ctx context.Context, customerID string, statusFilter string) ([]InvoiceSummaryDTO, error)
    // RenderPDF is deferred (not yet implemented)
    // RenderPDF(ctx context.Context, invoiceID string) (RenderedDocumentDTO, error)
}

type SessionService interface {
    Status(ctx context.Context) (SessionDTO, error)
    ValidateAccess(ctx context.Context) (bool, error)
}
```

---

## 10. Infrastructure Interfaces

These are practical dependencies required by the services.

```go
type Store interface {
    LegalEntities() LegalEntityStore
    CustomerProfiles() CustomerProfileStore
    IssuerProfiles() IssuerProfileStore
    Agreements() ServiceAgreementStore
    TimeEntries() TimeEntryStore
    Invoices() InvoiceStore
    Session() SessionStore
    InTx(ctx context.Context, fn func(ctx context.Context, store Store) error) error
}

type LegalEntityStore interface {
    Save(ctx context.Context, entity *LegalEntity) error
    GetByID(ctx context.Context, id string) (*LegalEntity, error)
    List(ctx context.Context, query ListQuery) (ListResult[LegalEntity], error)
    Delete(ctx context.Context, id string) error
}

type CustomerProfileStore interface {
    Save(ctx context.Context, profile *CustomerProfile) error
    GetByID(ctx context.Context, id string) (*CustomerProfile, error)
    List(ctx context.Context, query ListQuery) (ListResult[CustomerProfile], error)
    Delete(ctx context.Context, id string) error
}

type ServiceAgreementStore interface {
    Save(ctx context.Context, agreement *ServiceAgreement) error
    GetByID(ctx context.Context, id string) (*ServiceAgreement, error)
    ListByCustomerProfileID(ctx context.Context, customerProfileID string) ([]ServiceAgreement, error)
}

type TimeEntryStore interface {
    Save(ctx context.Context, entry *TimeEntry) error
    GetByID(ctx context.Context, id string) (*TimeEntry, error)
    ListUnbilledByCustomerProfile(ctx context.Context, customerProfileID string, period DateRange) ([]*TimeEntry, error)
}

type InvoiceStore interface {
    CreateDraft(ctx context.Context, invoice *Invoice, entries []*TimeEntry) error
    GetByID(ctx context.Context, id string) (*Invoice, error)
    Update(ctx context.Context, invoice *Invoice) error
    Delete(ctx context.Context, id string) error
    ListByCustomer(ctx context.Context, customerID string, status ...InvoiceStatus) ([]InvoiceSummary, error)
}

type IssuerProfileStore interface {
    Save(ctx context.Context, profile *IssuerProfile) error
    GetByID(ctx context.Context, id string) (*IssuerProfile, error)
}

type SessionStore interface {
    Save(ctx context.Context, session *Session) error
    GetCurrent(ctx context.Context) (*Session, error)
}

type IdentityVerifier interface {
    VerifyIDToken(ctx context.Context, rawToken string) (AuthenticatedIdentity, error)
}

type AccessPolicy interface {
    IsAllowed(email string) bool
}

type PDFRenderer interface {
    RenderInvoice(ctx context.Context, doc InvoiceDocument) ([]byte, error)
}
```

---

## 11. Commands / DTOs

```go
type CreateLegalEntityCommand struct {
    Type           string
    LegalName      string
    TradeName      string
    TaxID          string
    Email          string
    Phone          string
    Website        string
    BillingAddress AddressDTO
}
```

```go
type CreateCustomerProfileCommand struct {
    LegalEntityID   string
    DefaultCurrency string
    Notes           string
}
```

```go
type CreateIssuerProfileCommand struct {
    LegalEntityID   string
    DefaultCurrency string
    DefaultNotes    string
}
```

```go
type CreateServiceAgreementCommand struct {
    CustomerProfileID string
    Name         string
    Description  string
    BillingMode  string
    HourlyRate   int64
    Currency     string
    ValidFrom    *time.Time
    ValidUntil   *time.Time
}
```

```go
type RecordTimeEntryCommand struct {
    CustomerProfileID  string
    ServiceAgreementID string
    WorkDate           time.Time
    Hours              int64
    Description        string
    Billable           bool
}
```

```go
type CreateDraftInvoiceFromUnbilledTimeCommand struct {
    CustomerProfileID string
    From       time.Time
    To         time.Time
    Notes      string
}
```

```go
type HandleOAuthCallbackCommand struct {
    Code  string
    State string
}
```

```go
type AuthenticatedIdentity struct {
    Email          string
    EmailVerified  bool
    Subject        string
    Issuer         string
}
```

---

## 12. CLI Surface

```text
billar health
billar status

billar legal-entity create
billar legal-entity list
billar legal-entity get
billar legal-entity update
billar legal-entity delete

billar issuer create
billar issuer get
billar issuer update

billar customer create
billar customer list
billar customer get
billar customer update
billar customer delete
```

Planned downstream billing commands continue to reference the customer-facing role, now backed by `customer_profiles`:

```text
billar agreement create
billar agreement list --customer <customer-profile-id>
billar agreement rate-update --id <id>

billar time add
billar time list --customer <customer-profile-id>
billar time list-unbilled --customer <customer-profile-id>

billar invoice draft --customer <customer-profile-id> --from <date> --to <date>
billar invoice issue --id <id>
billar invoice discard --id <id>
billar invoice show --id <id>
billar invoice list --customer-id <customer-profile-id> [--status <status>]
billar invoice pdf --id <id>
```

---

## 13. MCP Tool Surface

* `session.status`
* `issuer_profile.create`
* `issuer_profile.get`
* `issuer_profile.update` — accepts inline legal entity fields (`type`, `legal_name`, `trade_name`, `tax_id`, `email`, `phone`, `website`, `billing_address`); cascades to the linked legal entity
* `customer_profile.create`
* `customer_profile.get`
* `customer_profile.list`
* `customer_profile.update` — accepts inline legal entity fields (`type`, `legal_name`, `trade_name`, `tax_id`, `email`, `phone`, `website`, `billing_address`); cascades to the linked legal entity
* `customer_profile.delete`
* `agreement.create`
* `agreement.list_by_customer_profile`
* `agreement.update_rate`
* `time_entry.create`
* `time_entry.list_by_customer_profile`
* `time_entry.list_unbilled`
* `invoice.draft`
* `invoice.issue`
* `invoice.discard`
* `invoice.get` — implemented (returns full `InvoiceDTO` with hydrated lines)
* `invoice.list` — implemented (returns `[]InvoiceSummaryDTO`, supports optional `status` filter)

**Deferred (future slices — not in current MCP surface)**:

* ~~`invoice.create_draft_from_unbilled`~~ — replaced by `invoice.draft` in current slice
* ~~`invoice.render_pdf`~~ — deferred; PDF rendering not yet wired

**Removed tools** (legal entity is always accessed via its owning profile, never directly):

* ~~`legal_entity.create`~~ — use `customer_profile.create` or `issuer_profile.create`
* ~~`legal_entity.get`~~ — profile responses include `legal_entity_id`; full legal entity fields are updated via `customer_profile.update` or `issuer_profile.update`
* ~~`legal_entity.list`~~ — list via profile tools
* ~~`legal_entity.update`~~ — use `customer_profile.update` or `issuer_profile.update`
* ~~`legal_entity.delete`~~ — legal entities are lifecycle-managed by their profiles

### Rule

MCP and CLI both call application services.
Neither should depend directly on SQLite or PDF libraries.
The HTTP MCP transport is exposed from the auth HTTP server at `/v1/mcp`.
Authentication for MCP-capable clients is owned by the Bearer API key middleware.
The current MCP session surface keeps only `session.status` for session inspection.
Early MCP bootstrap may include deterministic connector-level tools such as `hello_world` while the application surface is still growing.

---

## 14. Minimal Storage Model

Tables:

* `legal_entities`
* `customer_profiles`
* `issuer_profiles`
* `service_agreements`
* `time_entries`
* `invoices`
* `invoice_lines`
* `sessions`
* `invoice_sequences`

Rule:

* SQLite details stay inside infrastructure
* core logic does not know how persistence is implemented

---

## 15. Suggested Package Structure

```text
cmd/
  cli/
  mcp/

internal/
  core/
    legal_entity.go
    customer_profile.go
    service_agreement.go
    time_entry.go
    invoice.go
    issuer_profile.go
    session.go
    money.go
    hours.go
    types.go
    rules.go

  app/
    legal_entity_service.go
    customer_profile_service.go
    agreement_service.go
    time_entry_service.go
    invoice_service.go
    issuer_profile_service.go
    session_service.go
    legal_entity_dto.go
    customer_profile_dto.go
    issuer_profile_dto.go

  connectors/
    cli/
    mcp/
    mcphttp/

cmd/
  mcp-http/

  infra/
    sqlite/
    auth/
    pdf/
    config/
```

This structure is flatter, easier to read, and still keeps the right separation.

---

## 16. Suggested Libraries

These are candidate libraries, not hard requirements.

### HTTP / routing

* `net/http`
* `github.com/go-chi/chi/v5`

### OAuth / OIDC (removed from MCP HTTP path)

* ~~`github.com/coreos/go-oidc`~~ — no longer used for MCP HTTP auth
* ~~`golang.org/x/oauth2`~~ — no longer used for MCP HTTP auth

### SQLite

* `modernc.org/sqlite`

### PDF generation

Preferred direction:

* HTML template + browser-based PDF rendering

Candidates:

* `github.com/chromedp/chromedp`
* `github.com/SebastiaanKlippert/go-wkhtmltopdf`

Fallback if direct drawing is needed:

* `github.com/jung-kurt/gofpdf`

### Configuration

* environment variables first
* optional config file if needed later

### Notes

* there is no need for a heavy auth framework
* there is no need for a heavy DDD framework
* MCP compatibility comes from exposing the right HTTP/tool surface, not from a special billing library

---

## 17. Initial Milestone

Build the first end-to-end billing loop:

1. authenticate via Bearer API key (MCP HTTP) or use local CLI identity
2. establish authenticated access context
3. create legal entity for the issuer
4. configure issuer profile
5. create legal entity for the customer
6. create customer profile
7. create service agreement
8. record time entry
9. create draft invoice from unbilled time
10. issue invoice
11. render invoice PDF

That is the correct first slice.

---

## 18. Explicit Non-Goals For This Phase

Do not add yet:

* tax engine
* payments ledger
* exchange rates
* project/task hierarchy
* user accounts
* permissions
* SaaS multi-tenancy
* messaging/webhooks
* complex reporting
* custom authorization server

---

## 19. Remaining Decisions To Freeze

1. ~~invoice number format~~ → **Frozen**: `INV-YYYY-NNNN`, global yearly sequence persisted in `invoice_sequences` table, assigned only at issue time.
2. session timeout policy or manual logout-only
3. final PDF rendering library
4. invoice grouping policy for time entries
5. whether time entry editing is allowed after draft creation but before issue
6. ~~exact OAuth provider choice~~ → **Resolved**: MCP HTTP uses Bearer API key auth; CLI uses a local identity at startup.

Development default for the first runnable auth surface: Bearer API key for MCP HTTP; local identity for CLI.

---

## 20. Recommended Defaults

### Currency

* USD by default
* agreement and invoice must share currency
* no conversion

### Precision

* internal money precision: 4 digits
* internal hours precision: 4 digits
* invoice display preserves 4-digit formatting where needed

### Access

* API key authentication for MCP HTTP (Bearer token)
* MCP clients authenticate via Bearer API key; Billar middleware validates and injects identity
* CLI uses a local identity at startup
* single operator scope for now

### Invoice generation

* invoices generated from unbilled time entries
* issued invoices immutable
* rate snapshot preserved in invoice line

### PDF strategy

* prefer HTML-to-PDF
* preserve the current invoice layout as closely as practical

---

## 21. Final Cut

The minimum stable internal model is:

* `LegalEntity`
* `CustomerProfile`
* `ServiceAgreement`
* `TimeEntry`
* `Invoice`
* `InvoiceLine`
* `IssuerProfile`
* `Session`
* shared types: `Money`, `Hours`, `CurrencyCode`, `InvoiceNumber`, `Address`

The minimum viable stack is:

* Go
* SQLite
* Chi or net/http
* HTML-to-PDF renderer

This is enough to build the first serious version without forcing full DDD ceremony or premature infrastructure complexity.
