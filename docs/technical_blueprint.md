# Dev Invoice App — Technical Blueprint (Initial Domain Baseline)

## 1. Scope

This system is a Go modular monolith for managing:

- customers
- customer-specific service agreements
- billable time entries
- invoice generation
- invoice PDF rendering
- CLI execution
- MCP tool execution
- local database unlock/session access

This baseline intentionally stays simple.

It does **not** include:

- tax engine
- multi-user auth
- multi-tenant support
- CQRS
- event sourcing
- message broker
- workflow engine
- distributed services

The main objective is to support a reliable hourly billing flow with a clean domain and stable boundaries.

---

## 2. Frozen Decisions

### Architecture style
- Clean DDD + Hexagonal
- Trimmed for a small, practical Go application
- Strict inward dependency rule
- Modular monolith

### Persistence
- SQLite only
- Full-file encryption
- No Turso
- No cloud sync in the initial version

### Access model
- Single local operator
- Session/unlock gate required before DB access
- Session unlock uses a local secret/passphrase

### Billing model
- Hourly billing first
- Customer can have one or more service agreements
- One service agreement has one hourly rate at a time
- Invoice quantity is total billable hours

### Currency
- USD default
- Base support for future multi-currency
- No exchange-rate logic yet
- No FX conversion yet

### Precision
- Money stored as integer
- Hours stored as integer
- Money precision: 4 digits
- Hours precision: 4 digits
- Invoice display preserves 4-digit precision where required

### Tax policy
- No tax calculation in the initial version
- Model leaves room for future tax support

### PDF rendering
- Use the library/tooling that gives the easiest path to reproduce the current invoice layout
- Rendering remains an infrastructure concern behind a port

---

## 3. Design Principles

1. Domain owns business rules.
2. Application layer orchestrates use cases.
3. Adapters handle external concerns.
4. Repositories are per aggregate, not per table.
5. Entities contain behavior, not just data.
6. Value objects are immutable and self-validating.
7. Do not expose persistence directly to CLI or MCP.
8. MCP and CLI call application use cases only.
9. Start with the smallest domain that fully supports billing.
10. Preserve invoice reproducibility: issued invoices must not drift.

---

## 4. Bounded Contexts

## 4.1 Billing
Core domain.

Contains:
- Customer
- ServiceAgreement
- TimeEntry
- Invoice
- InvoiceLine
- IssuerProfile
- Money
- Hours
- CurrencyCode

## 4.2 Access
Local unlock/session boundary.

Contains:
- Session
- Unlock policy
- Secret verification
- Lock/unlock lifecycle

## 4.3 Document Rendering
Support context.

Contains:
- invoice document projection
- template selection
- PDF rendering

This context does not own billing rules.

---

## 5. Core Domain Model

## 5.1 Aggregate: Customer

Represents the billed party.

### Entity: `Customer`
Fields:
- `CustomerID`
- `CustomerType` (`Individual`, `Company`)
- `LegalName`
- `TradeName`
- `TaxID`
- `Email`
- `Phone`
- `Website`
- `BillingAddress`
- `Status`
- `DefaultCurrency`
- `Notes`
- `CreatedAt`
- `UpdatedAt`

### Value objects used
- `CustomerID`
- `CustomerType`
- `TaxIdentifier`
- `EmailAddress`
- `PhoneNumber`
- `Address`
- `CustomerStatus`
- `CurrencyCode`

### Behavior
- create customer
- update billing identity
- update contact info
- activate/deactivate
- validate invoicing readiness

---

## 5.2 Aggregate: Service Agreement

Represents the billing agreement between operator and customer.

### Entity: `ServiceAgreement`
Fields:
- `ServiceAgreementID`
- `CustomerID`
- `Code`
- `Name`
- `Description`
- `BillingMode`
- `HourlyRate`
- `Currency`
- `Active`
- `ValidFrom`
- `ValidUntil`
- `CreatedAt`
- `UpdatedAt`

### Value objects used
- `ServiceAgreementID`
- `CustomerID`
- `BillingMode`
- `Money`
- `CurrencyCode`
- `DateRange`

### Behavior
- create service agreement
- change hourly rate
- activate/deactivate
- check if billable on date
- enforce billing mode constraints

### Notes
- Initial version supports `hourly` billing mode only.
- Model should keep `BillingMode` extensible for future fixed-fee or milestone billing.

---

## 5.3 Aggregate: Time Entry

Represents recorded work.

### Entity: `TimeEntry`
Fields:
- `TimeEntryID`
- `CustomerID`
- `ServiceAgreementID`
- `WorkDate`
- `Hours`
- `Description`
- `Billable`
- `Invoiced`
- `InvoiceID` (optional)
- `CreatedAt`
- `UpdatedAt`

### Value objects used
- `TimeEntryID`
- `CustomerID`
- `ServiceAgreementID`
- `Hours`
- `WorkDate`

### Behavior
- record time entry
- update description
- mark non-billable
- assign to invoice
- prevent double billing
- validate agreement/date compatibility

---

## 5.4 Aggregate: Invoice

Represents a financial document issued to a customer.

### Aggregate root: `Invoice`
Fields:
- `InvoiceID`
- `InvoiceNumber`
- `CustomerID`
- `IssueDate`
- `DueDate`
- `Currency`
- `Status`
- `Notes`
- `Subtotal`
- `GrandTotal`
- `Lines`
- `CreatedAt`
- `UpdatedAt`

### Child entity: `InvoiceLine`
Fields:
- `InvoiceLineID`
- `ServiceAgreementID` (optional)
- `Description`
- `Quantity`
- `UnitPrice`
- `LineSubtotal`
- `LineTotal`
- `SourceType`
- `SourceRef`
- `SortOrder`

### Value objects used
- `InvoiceID`
- `InvoiceLineID`
- `InvoiceNumber`
- `InvoiceStatus`
- `CurrencyCode`
- `Money`
- `Quantity`

### Behavior
- create draft invoice
- add/remove line while draft
- recalculate totals
- issue invoice
- void invoice
- prevent edits after issue
- preserve monetary snapshot

### Notes
- No tax fields in the first version.
- Structure should still allow future extension without breaking aggregate semantics.

---

## 5.5 Aggregate: Issuer Profile

Represents the operator/company issuing invoices.

### Entity: `IssuerProfile`
Fields:
- `IssuerProfileID`
- `LegalName`
- `TaxID`
- `Email`
- `Phone`
- `Website`
- `Address`
- `DefaultCurrency`
- `InvoicePrefix`
- `PaymentInstructions`
- `CreatedAt`
- `UpdatedAt`

### Behavior
- update issuer identity
- update invoice defaults
- validate render readiness

---

## 5.6 Aggregate: Session

Represents local DB access state.

### Entity: `Session`
Fields:
- `SessionID`
- `Unlocked`
- `UnlockedAt`
- `LastActivityAt`
- `LockedAt`

### Behavior
- unlock
- lock
- validate active access
- expire/invalidate if policy requires it later

### Notes
- This is application-facing security state, not enterprise identity.
- Single local operator means no account model is needed now.

---

## 6. Shared Value Objects

## 6.1 `Money`
Represents monetary value.

### Rules
- stored as integer
- precision fixed to 4 digits
- always carries `CurrencyCode`
- immutable
- no float usage

### Suggested shape
- `Amount int64`
- `Currency CurrencyCode`
- interpretation: scaled integer with 4 decimal digits

Example:
- `156250` => `15.6250`
- `25000000` => `2500.0000`

### Operations
- add
- subtract
- multiply by quantity/hours
- compare
- format for invoice display

---

## 6.2 `Hours`
Represents billable time quantity.

### Rules
- stored as integer
- precision fixed to 4 digits
- immutable
- no float usage

### Suggested shape
- `Value int64`
- interpretation: scaled integer with 4 decimal digits

Example:
- `1600000` => `160.0000`
- `12500` => `1.2500`

### Operations
- add
- compare
- validate positive
- format for invoice display

---

## 6.3 Other value objects
- `CurrencyCode`
- `InvoiceNumber`
- `Address`
- `EmailAddress`
- `PhoneNumber`
- `TaxIdentifier`
- `DateRange`
- `CustomerStatus`
- `InvoiceStatus`
- `BillingMode`

---

## 7. Business Rules

## 7.1 Customer rules
1. A customer must have a billing name.
2. A customer may exist without any service agreement.
3. A customer may be deactivated.
4. Inactive customers should not receive new invoices unless explicitly allowed by future admin policy.
5. Default currency is USD unless explicitly set otherwise.

## 7.2 Service agreement rules
1. A service agreement belongs to exactly one customer.
2. Initial version only supports `hourly` billing mode.
3. Hourly rate must be positive.
4. Agreement currency defaults to USD.
5. Agreement must be active to accept billable time entries.
6. Rate changes must not mutate already issued invoices.
7. Agreement may have an optional validity window.

## 7.3 Time entry rules
1. Hours must be greater than zero.
2. Billable time must reference an active service agreement.
3. Time entry date must fit the agreement validity window if one exists.
4. A time entry may be linked to one invoice only.
5. Once invoiced, the entry is financially locked.
6. Non-billable entries must not be included in invoice generation.

## 7.4 Invoice rules
1. An invoice must have at least one line.
2. Invoice totals are always derived from lines.
3. Invoice currency must be internally consistent.
4. A draft invoice is editable.
5. An issued invoice is immutable.
6. A time entry cannot be billed more than once.
7. Invoice numbering must be unique and sequential.
8. Line totals are computed from quantity and unit price.
9. Grand total equals subtotal in the initial version because tax is disabled.
10. Issued invoices preserve pricing snapshots.

## 7.5 Session/access rules
1. No repository access before unlock.
2. Unlock requires the operator secret/passphrase.
3. Lock invalidates persistence access.
4. CLI and MCP commands that need data access must fail fast if session is locked.
5. Encryption details stay outside domain logic.

---

## 8. Aggregate Boundaries

### Customer
Separate aggregate.
Referenced by ID.

### ServiceAgreement
Separate aggregate.
Referenced by ID.

### TimeEntry
Separate aggregate.
Referenced by ID.

### Invoice
Separate aggregate with `InvoiceLine` inside it.
Invoice and lines must stay transactionally consistent together.

### IssuerProfile
Separate aggregate.

### Session
Separate aggregate/application boundary object.

### Rule
Cross-aggregate references are by ID only.

---

## 9. Domain Services

Only add domain services where behavior does not fit naturally inside one entity.

## 9.1 `InvoiceFactory`
Purpose:
- create draft invoice from unbilled time entries
- group time entries according to invoice policy
- produce invoice lines using agreement snapshots

## 9.2 `InvoiceNumberPolicy`
Purpose:
- define invoice numbering format
- validate generated numbers

## 9.3 `CurrencyPolicy`
Purpose:
- enforce current currency constraints
- keep room for future multi-currency rules

### Current behavior
- all generated invoices use USD by default
- agreement currency and invoice currency must match in the initial version
- no FX conversion supported

---

## 10. Application Use Cases

## 10.1 Access
- `InitializeVault`
- `UnlockSession`
- `LockSession`
- `GetSessionStatus`
- `RotateUnlockSecret`

## 10.2 Customer
- `CreateCustomer`
- `UpdateCustomer`
- `GetCustomer`
- `ListCustomers`
- `DeactivateCustomer`

## 10.3 Service Agreement
- `CreateServiceAgreement`
- `UpdateServiceAgreementRate`
- `ActivateServiceAgreement`
- `DeactivateServiceAgreement`
- `ListCustomerServiceAgreements`

## 10.4 Time Entry
- `RecordTimeEntry`
- `UpdateTimeEntry`
- `DeleteDraftTimeEntry`
- `ListCustomerTimeEntries`
- `ListUnbilledTimeEntries`

## 10.5 Invoice
- `CreateDraftInvoice`
- `CreateDraftInvoiceFromUnbilledTime`
- `AddManualInvoiceLine`
- `RemoveDraftInvoiceLine`
- `RecalculateInvoiceTotals`
- `IssueInvoice`
- `VoidInvoice`
- `GetInvoice`
- `ListInvoices`
- `RenderInvoicePDF`

## 10.6 Issuer Profile
- `SetupIssuerProfile`
- `UpdateIssuerProfile`
- `GetIssuerProfile`

---

## 11. Driver Ports

These are the interfaces exposed to CLI and MCP.

```go
type UnlockSessionUseCase interface {
    Execute(ctx context.Context, cmd UnlockSessionCommand) (SessionDTO, error)
}

type CreateCustomerUseCase interface {
    Execute(ctx context.Context, cmd CreateCustomerCommand) (CustomerDTO, error)
}

type CreateServiceAgreementUseCase interface {
    Execute(ctx context.Context, cmd CreateServiceAgreementCommand) (ServiceAgreementDTO, error)
}

type RecordTimeEntryUseCase interface {
    Execute(ctx context.Context, cmd RecordTimeEntryCommand) (TimeEntryDTO, error)
}

type CreateDraftInvoiceFromUnbilledTimeUseCase interface {
    Execute(ctx context.Context, cmd CreateDraftInvoiceFromUnbilledTimeCommand) (InvoiceDTO, error)
}

type IssueInvoiceUseCase interface {
    Execute(ctx context.Context, cmd IssueInvoiceCommand) (InvoiceDTO, error)
}

type RenderInvoicePDFUseCase interface {
    Execute(ctx context.Context, query RenderInvoicePDFQuery) (RenderedDocumentDTO, error)
}
```

---

## 12. Driven Ports

These are the interfaces application/domain depend on.

```go
type CustomerRepository interface {
    Save(ctx context.Context, customer *billing.Customer) error
    GetByID(ctx context.Context, id billing.CustomerID) (*billing.Customer, error)
    List(ctx context.Context, filter CustomerFilter) ([]*billing.Customer, error)
}

type ServiceAgreementRepository interface {
    Save(ctx context.Context, agreement *billing.ServiceAgreement) error
    GetByID(ctx context.Context, id billing.ServiceAgreementID) (*billing.ServiceAgreement, error)
    ListByCustomerID(ctx context.Context, customerID billing.CustomerID) ([]*billing.ServiceAgreement, error)
}

type TimeEntryRepository interface {
    Save(ctx context.Context, entry *billing.TimeEntry) error
    GetByID(ctx context.Context, id billing.TimeEntryID) (*billing.TimeEntry, error)
    ListUnbilledByCustomer(ctx context.Context, customerID billing.CustomerID, period billing.DateRange) ([]*billing.TimeEntry, error)
}

type InvoiceRepository interface {
    Save(ctx context.Context, invoice *billing.Invoice) error
    GetByID(ctx context.Context, id billing.InvoiceID) (*billing.Invoice, error)
    NextInvoiceNumber(ctx context.Context) (billing.InvoiceNumber, error)
}

type IssuerProfileRepository interface {
    Save(ctx context.Context, profile *billing.IssuerProfile) error
    Get(ctx context.Context) (*billing.IssuerProfile, error)
}

type SessionRepository interface {
    Save(ctx context.Context, session *access.Session) error
    GetCurrent(ctx context.Context) (*access.Session, error)
}

type UnlockSecretVerifier interface {
    Verify(ctx context.Context, secret string) error
}

type PDFRenderer interface {
    RenderInvoice(ctx context.Context, doc InvoiceDocument) ([]byte, error)
}

type UnitOfWork interface {
    Do(ctx context.Context, fn func(ctx context.Context) error) error
}
```

---

## 13. Command / Query DTO Sketches

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
type IssueInvoiceCommand struct {
    InvoiceID string
}
```

---

## 14. CLI Surface

```text
invoice unlock
invoice lock
invoice session status

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

## 15. MCP Tool Surface

- `session.unlock`
- `session.lock`
- `session.status`
- `issuer.setup`
- `issuer.get`
- `customer.create`
- `customer.get`
- `customer.list`
- `agreement.create`
- `agreement.list_by_customer`
- `agreement.update_rate`
- `time_entry.create`
- `time_entry.list_by_customer`
- `time_entry.list_unbilled`
- `invoice.create_draft_from_unbilled`
- `invoice.get`
- `invoice.list`
- `invoice.issue`
- `invoice.render_pdf`

### Constraint
MCP never talks to repositories directly.
MCP calls application use cases only.

---

## 16. Storage Notes

### Minimal tables
- `customers`
- `service_agreements`
- `time_entries`
- `invoices`
- `invoice_lines`
- `issuer_profile`
- `sessions`
- `invoice_sequences`

### Storage boundary rule
Encryption, file unlock mechanics, and SQLite specifics stay in adapters/infrastructure.
The domain does not know encryption exists.

---

## 17. Suggested Package Structure

```text
cmd/
  cli/
  mcp/

internal/
  domain/
    customer/
      entity.go
      value_objects.go
      repository.go
      errors.go
    serviceagreement/
      entity.go
      value_objects.go
      repository.go
      errors.go
    timeentry/
      entity.go
      value_objects.go
      repository.go
      errors.go
    invoice/
      entity.go
      line.go
      value_objects.go
      repository.go
      services.go
      errors.go
    issuer/
      entity.go
      repository.go
    access/
      session.go
      repository.go
      errors.go
    shared/
      money.go
      hours.go
      address.go
      currency.go
      ids.go
      daterange.go

  application/
    access/
    customer/
    serviceagreement/
    timeentry/
    invoice/
    issuer/
    shared/
      uow.go
      dto.go

  infrastructure/
    persistence/
      sqlite/
    security/
      vault/
    rendering/
      pdf/
    cli/
    mcp/
    config/
    di/
```

---

## 18. Initial Milestone

Build the first end-to-end billing loop:

1. unlock session
2. configure issuer profile
3. create customer
4. create service agreement
5. record time entry
6. create draft invoice from unbilled time
7. issue invoice
8. render invoice PDF

That is the correct first slice.

---

## 19. Explicit Non-Goals For This Phase

Do not add yet:
- tax engine
- payments ledger
- exchange rates
- project/task hierarchy
- user accounts
- permissions
- SaaS multi-tenancy
- messaging/webhooks
- complex reporting

---

## 20. Next Decisions To Freeze

1. invoice number format
2. session timeout policy or manual lock-only
3. PDF rendering library/tool selection
4. whether invoice lines are grouped by service agreement only or also by date/description
5. whether time entry editing is allowed after draft invoice creation but before issue

---

## 21. Recommended Default Policies

### Currency policy
- USD by default
- agreement and invoice must share currency
- no conversion

### Precision policy
- internal money precision: 4 digits
- internal hours precision: 4 digits
- invoice display: preserve 4-digit formatting for rate and quantity where needed

### Access policy
- manual unlock required
- manual lock available
- single operator only

### Invoice policy
- invoices generated from unbilled time entries
- issued invoices immutable
- rate snapshot preserved in invoice line

---

## 22. Final Domain Cut

The minimum stable domain is:

- `Customer`
- `ServiceAgreement`
- `TimeEntry`
- `Invoice`
- `InvoiceLine`
- `IssuerProfile`
- `Session`
- shared value objects: `Money`, `Hours`, `CurrencyCode`, `InvoiceNumber`, `Address`

This is enough to build the first serious version without architecture bloat.

