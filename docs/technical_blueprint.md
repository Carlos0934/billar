# Dev Invoice App — Technical Blueprint (Pragmatic Baseline)

## 1. Scope

This system is a Go application for managing:

* customers
* customer-specific service agreements
* billable time entries
* invoice generation
* invoice PDF rendering
* CLI execution
* MCP tool execution
* local session-based access

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
* OAuth/OIDC-based authentication
* Allow access by exact email allowlist and allowed email domains
* Session gate required before protected operations
* Session stores authenticated user identity

### Billing model

* Hourly billing first
* Customer can have one or more service agreements
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

* customer
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

* create customer
* create agreement
* record time entry
* create invoice draft
* issue invoice
* render invoice pdf
* inspect current session state
* handle auth callback/runtime for development and testing
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
* OAuth/OIDC integration
* session persistence
* PDF renderer
* config loading
* local HTTP auth server for OAuth/OIDC callback handling during development

---

## 6. Core Data Model

The app only needs a few stable business objects.

## 6.1 Customer

Represents the billed party.

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
* `Status`
* `DefaultCurrency`
* `Notes`
* `CreatedAt`
* `UpdatedAt`

### Rules

* a customer must have a billing name
* inactive customers should not receive new invoices
* default currency is USD unless explicitly set otherwise

---

## 6.2 ServiceAgreement

Represents a billable agreement for a customer.

### Fields

* `ID`
* `CustomerID`
* `Code`
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

* belongs to exactly one customer
* initial version supports `hourly` only
* hourly rate must be positive
* agreement must be active to record billable time
* rate changes must not mutate issued invoices

---

## 6.3 TimeEntry

Represents recorded work.

### Fields

* `ID`
* `CustomerID`
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

## 6.4 Invoice

Represents a financial document.

### Fields

* `ID`
* `InvoiceNumber`
* `CustomerID`
* `IssueDate`
* `DueDate`
* `Currency`
* `Status`
* `Notes`
* `Subtotal`
* `GrandTotal`
* `Lines`
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

---

## 6.5 IssuerProfile

Represents the operator/company issuing invoices.

### Fields

* `ID`
* `LegalName`
* `TaxID`
* `Email`
* `Phone`
* `Website`
* `Address`
* `DefaultCurrency`
* `InvoicePrefix`
* `PaymentInstructions`
* `CreatedAt`
* `UpdatedAt`

---

## 6.6 Session

Represents current access state.

### Fields

* `ID`
* `UserEmail`
* `Unlogouted`
* `UnlogoutedAt`
* `LastActivityAt`
* `LockedAt`

### Rules

* no protected operation before session login
* login requires valid OAuth/OIDC identity
* only allowed emails or allowed domains can authenticate session
* logout invalidates protected access
* CLI and MCP operations that need storage must fail fast if logouted

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
* `CustomerStatus`
* `InvoiceStatus`
* `BillingMode`

---

## 8. Application Services

Keep the app surface small and explicit.

## 8.1 Access

* `StartLogin`
* `HandleOAuthCallback`
* `AuthenticateSession`
* `Logout`
* `GetSessionStatus`
* `ValidateAccess`

## 8.2 Customer

* `CreateCustomer`
* `UpdateCustomer`
* `GetCustomer`
* `ListCustomers`
* `DeactivateCustomer`

## 8.3 Service Agreement

* `CreateServiceAgreement`
* `UpdateServiceAgreementRate`
* `ActivateServiceAgreement`
* `DeactivateServiceAgreement`
* `ListCustomerServiceAgreements`

## 8.4 Time Entry

* `RecordTimeEntry`
* `UpdateTimeEntry`
* `DeleteDraftTimeEntry`
* `ListCustomerTimeEntries`
* `ListUnbilledTimeEntries`

## 8.5 Invoice

* `CreateDraftInvoice`
* `CreateDraftInvoiceFromUnbilledTime`
* `AddManualInvoiceLine`
* `RemoveDraftInvoiceLine`
* `RecalculateInvoiceTotals`
* `IssueInvoice`
* `VoidInvoice`
* `GetInvoice`
* `ListInvoices`
* `RenderInvoicePDF`

## 8.6 Issuer Profile

* `SetupIssuerProfile`
* `UpdateIssuerProfile`
* `GetIssuerProfile`

---

## 9. Connector Interfaces

These are enough. No need to overname them as formal ports everywhere.

```go
type CustomerService interface {
    Create(ctx context.Context, cmd CreateCustomerCommand) (CustomerDTO, error)
    Update(ctx context.Context, cmd UpdateCustomerCommand) (CustomerDTO, error)
    Get(ctx context.Context, id string) (CustomerDTO, error)
    List(ctx context.Context, filter CustomerFilter) ([]CustomerDTO, error)
}

type AgreementService interface {
    Create(ctx context.Context, cmd CreateServiceAgreementCommand) (ServiceAgreementDTO, error)
    UpdateRate(ctx context.Context, cmd UpdateServiceAgreementRateCommand) (ServiceAgreementDTO, error)
    ListByCustomer(ctx context.Context, customerID string) ([]ServiceAgreementDTO, error)
}

type TimeEntryService interface {
    Record(ctx context.Context, cmd RecordTimeEntryCommand) (TimeEntryDTO, error)
    ListUnbilled(ctx context.Context, customerID string, period DateRangeDTO) ([]TimeEntryDTO, error)
}

type InvoiceService interface {
    CreateDraftFromUnbilled(ctx context.Context, cmd CreateDraftInvoiceFromUnbilledTimeCommand) (InvoiceDTO, error)
    Issue(ctx context.Context, invoiceID string) (InvoiceDTO, error)
    Get(ctx context.Context, invoiceID string) (InvoiceDTO, error)
    RenderPDF(ctx context.Context, invoiceID string) (RenderedDocumentDTO, error)
}

type SessionService interface {
    StartLogin(ctx context.Context) (LoginURLDTO, error)
    HandleOAuthCallback(ctx context.Context, cmd HandleOAuthCallbackCommand) (SessionDTO, error)
    AuthenticateSession(ctx context.Context, rawToken string) (SessionDTO, error)
    Logout(ctx context.Context) error
    Status(ctx context.Context) (SessionDTO, error)
    ValidateAccess(ctx context.Context) (bool, error)
}
```

---

## 10. Infrastructure Interfaces

These are practical dependencies required by the services.

```go
type Store interface {
    Customers() CustomerStore
    Agreements() ServiceAgreementStore
    TimeEntries() TimeEntryStore
    Invoices() InvoiceStore
    IssuerProfile() IssuerProfileStore
    Session() SessionStore
    InTx(ctx context.Context, fn func(ctx context.Context, store Store) error) error
}

type CustomerStore interface {
    Save(ctx context.Context, customer *Customer) error
    GetByID(ctx context.Context, id string) (*Customer, error)
    List(ctx context.Context, filter CustomerFilter) ([]*Customer, error)
}

type ServiceAgreementStore interface {
    Save(ctx context.Context, agreement *ServiceAgreement) error
    GetByID(ctx context.Context, id string) (*ServiceAgreement, error)
    ListByCustomerID(ctx context.Context, customerID string) ([]*ServiceAgreement, error)
}

type TimeEntryStore interface {
    Save(ctx context.Context, entry *TimeEntry) error
    GetByID(ctx context.Context, id string) (*TimeEntry, error)
    ListUnbilledByCustomer(ctx context.Context, customerID string, period DateRange) ([]*TimeEntry, error)
}

type InvoiceStore interface {
    Save(ctx context.Context, invoice *Invoice) error
    GetByID(ctx context.Context, id string) (*Invoice, error)
    NextInvoiceNumber(ctx context.Context) (string, error)
}

type IssuerProfileStore interface {
    Save(ctx context.Context, profile *IssuerProfile) error
    Get(ctx context.Context) (*IssuerProfile, error)
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
type CreateCustomerCommand struct {
    CustomerType string
    LegalName    string
    TradeName    string
    TaxID        string
    Email        string
    Phone        string
    Website      string
    Currency     string
    Address      AddressDTO
    Notes        string
}
```

```go
type CreateServiceAgreementCommand struct {
    CustomerID   string
    Code         string
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
    CustomerID         string
    ServiceAgreementID string
    WorkDate           time.Time
    Hours              int64
    Description        string
    Billable           bool
}
```

```go
type CreateDraftInvoiceFromUnbilledTimeCommand struct {
    CustomerID string
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
invoice auth login
invoice auth logout
invoice auth status

invoice issuer setup
invoice issuer show

invoice customer create
invoice customer list
invoice customer get

invoice agreement create
invoice agreement list --customer <id>
invoice agreement rate-update --id <id>

invoice time add
invoice time list --customer <id>
invoice time list-unbilled --customer <id>

invoice invoice draft --customer <id> --from <date> --to <date>
invoice invoice issue --id <id>
invoice invoice show --id <id>
invoice invoice pdf --id <id>
```

---

## 13. MCP Tool Surface

* `session.status`
* `issuer.setup`
* `issuer.get`
* `customer.create`
* `customer.get`
* `customer.list`
* `agreement.create`
* `agreement.list_by_customer`
* `agreement.update_rate`
* `time_entry.create`
* `time_entry.list_by_customer`
* `time_entry.list_unbilled`
* `invoice.create_draft_from_unbilled`
* `invoice.get`
* `invoice.list`
* `invoice.issue`
* `invoice.render_pdf`

### Rule

MCP and CLI both call application services.
Neither should depend directly on SQLite, OAuth internals, or PDF libraries.
OAuth callback handling is HTTP-only and does not belong in the MCP tool surface.
The HTTP MCP transport is exposed from the auth HTTP server at `/v1/mcp`.
Authentication for MCP-capable clients is owned by the client, not by Billar tools.
The current MCP session surface keeps only `session.status` for session inspection.
Stdio remains available as a technical development and testing transport, not as the real interactive authentication UX.
Early MCP bootstrap may include deterministic connector-level tools such as `hello_world` while the application surface is still growing.
For MCP ingress, exact email allowlist or allowed domain remains required, and IP allowlisting is optional by config.

---

## 14. Minimal Storage Model

Tables:

* `customers`
* `service_agreements`
* `time_entries`
* `invoices`
* `invoice_lines`
* `issuer_profile`
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
    customer.go
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
    customer_service.go
    agreement_service.go
    time_entry_service.go
    invoice_service.go
    issuer_service.go
    session_service.go
    dto.go
    commands.go

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

### OAuth / OIDC

* `github.com/coreos/go-oidc`
* `golang.org/x/oauth2`

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

1. authenticate/login
2. establish session
3. configure issuer profile
4. create customer
5. create service agreement
6. record time entry
7. create draft invoice from unbilled time
8. issue invoice
9. render invoice PDF

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

1. invoice number format
2. session timeout policy or manual logout-only
3. final PDF rendering library
4. invoice grouping policy for time entries
5. whether time entry editing is allowed after draft creation but before issue
6. exact OAuth provider choice

Development default for the first runnable auth surface: Google OIDC with a loopback HTTP callback.

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

* OAuth/OIDC login required
* allow exact email matches and allowed domains
* MCP may additionally enforce optional ingress IP allowlists by config
* MCP clients own their interactive auth flow; Billar exposes session inspection and preserves the HTTP callback/runtime for development and testing
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

* `Customer`
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
* go-oidc + oauth2
* HTML-to-PDF renderer

This is enough to build the first serious version without forcing full DDD ceremony or premature infrastructure complexity.
