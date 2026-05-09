package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Carlos0934/billar/internal/app"
	"github.com/Carlos0934/billar/internal/core"
)

type QuoteStore struct {
	db *sql.DB
}

func NewQuoteStore(store *Store) *QuoteStore {
	if store == nil {
		return nil
	}
	return &QuoteStore{db: store.DB()}
}

func (s *QuoteStore) Create(ctx context.Context, quote *core.Quote) error {
	if s == nil || s.db == nil {
		return errors.New("quote sqlite store is required")
	}
	if quote == nil {
		return errors.New("quote is required")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()
	if err := insertQuote(ctx, tx, quote); err != nil {
		return err
	}
	for _, line := range quote.Lines {
		if err := insertQuoteLine(ctx, tx, quote.ID, line, quote.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *QuoteStore) GetByID(ctx context.Context, id string) (*core.Quote, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("quote sqlite store is required")
	}
	var quote core.Quote
	var status string
	var sentAt, acceptedAt, rejectedAt, expiredAt, createdAt, updatedAt sql.NullInt64
	err := s.db.QueryRowContext(ctx, `
SELECT id, customer_id, status, currency, notes, sent_at, accepted_at, rejected_at, expired_at, created_at, updated_at
FROM quotes WHERE id = ?`, strings.TrimSpace(id)).Scan(&quote.ID, &quote.CustomerID, &status, &quote.Currency, &quote.Notes, &sentAt, &acceptedAt, &rejectedAt, &expiredAt, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, app.ErrQuoteNotFound
		}
		return nil, fmt.Errorf("get quote by id: %w", err)
	}
	quote.Status = core.QuoteStatus(status)
	quote.SentAt = nanoToTime(sentAt)
	quote.AcceptedAt = nanoToTime(acceptedAt)
	quote.RejectedAt = nanoToTime(rejectedAt)
	quote.ExpiredAt = nanoToTime(expiredAt)
	quote.CreatedAt = nanoToTime(createdAt)
	quote.UpdatedAt = nanoToTime(updatedAt)
	lines, err := s.linesByQuote(ctx, quote.ID)
	if err != nil {
		return nil, err
	}
	quote.Lines = lines
	return &quote, nil
}

func (s *QuoteStore) ListByCustomer(ctx context.Context, customerID string, statuses ...core.QuoteStatus) ([]core.QuoteSummary, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("quote sqlite store is required")
	}
	args := []interface{}{strings.TrimSpace(customerID)}
	query := `
SELECT q.id, q.customer_id, q.status, q.currency, q.created_at,
       COALESCE(SUM(ql.quantity_min * ql.unit_rate_amount / 60), 0) AS total_amount
FROM quotes q
LEFT JOIN quote_lines ql ON ql.quote_id = q.id
WHERE q.customer_id = ?`
	if len(statuses) > 0 {
		query += ` AND q.status = ?`
		args = append(args, string(statuses[0]))
	}
	query += ` GROUP BY q.id, q.customer_id, q.status, q.currency, q.created_at ORDER BY q.created_at DESC, q.id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list quotes by customer: %w", err)
	}
	defer rows.Close()
	var summaries []core.QuoteSummary
	for rows.Next() {
		var summary core.QuoteSummary
		var status string
		var createdAt sql.NullInt64
		var totalAmount int64
		if err := rows.Scan(&summary.ID, &summary.CustomerID, &status, &summary.Currency, &createdAt, &totalAmount); err != nil {
			return nil, fmt.Errorf("scan quote summary: %w", err)
		}
		summary.Status = core.QuoteStatus(status)
		summary.CreatedAt = nanoToTime(createdAt)
		summary.Total = core.Money{Amount: totalAmount, Currency: summary.Currency}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quote summaries: %w", err)
	}
	return summaries, nil
}

func (s *QuoteStore) AddLine(ctx context.Context, quoteID string, line core.QuoteLine) error {
	if s == nil || s.db == nil {
		return errors.New("quote sqlite store is required")
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO quote_lines (id, quote_id, service_agreement_id, description, quantity_min, unit_rate_amount, unit_rate_currency, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, line.ID, strings.TrimSpace(quoteID), line.ServiceAgreementID, line.Description, line.QuantityMin, line.UnitRate.Amount, line.UnitRate.Currency, time.Now().UTC().UnixNano())
	if err != nil {
		return fmt.Errorf("insert quote line: %w", err)
	}
	return nil
}

func (s *QuoteStore) Update(ctx context.Context, quote *core.Quote) error {
	if s == nil || s.db == nil {
		return errors.New("quote sqlite store is required")
	}
	if quote == nil {
		return errors.New("quote is required")
	}
	result, err := s.db.ExecContext(ctx, `
UPDATE quotes SET status = ?, notes = ?, sent_at = ?, accepted_at = ?, rejected_at = ?, expired_at = ?, updated_at = ? WHERE id = ?`,
		string(quote.Status), quote.Notes, timeToNano(quote.SentAt), timeToNano(quote.AcceptedAt), timeToNano(quote.RejectedAt), timeToNano(quote.ExpiredAt), timeToNano(quote.UpdatedAt), quote.ID)
	if err != nil {
		return fmt.Errorf("update quote: %w", err)
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return app.ErrQuoteNotFound
	}
	return nil
}

func (s *QuoteStore) Delete(ctx context.Context, id string) error {
	if s == nil || s.db == nil {
		return errors.New("quote sqlite store is required")
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM quotes WHERE id = ?`, strings.TrimSpace(id))
	if err != nil {
		return fmt.Errorf("delete quote: %w", err)
	}
	if rows, err := result.RowsAffected(); err == nil && rows == 0 {
		return app.ErrQuoteNotFound
	}
	return nil
}

func insertQuote(ctx context.Context, tx *sql.Tx, quote *core.Quote) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO quotes (id, customer_id, status, currency, notes, sent_at, accepted_at, rejected_at, expired_at, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, quote.ID, quote.CustomerID, string(quote.Status), quote.Currency, quote.Notes, timeToNano(quote.SentAt), timeToNano(quote.AcceptedAt), timeToNano(quote.RejectedAt), timeToNano(quote.ExpiredAt), quote.CreatedAt.UTC().UnixNano(), quote.UpdatedAt.UTC().UnixNano())
	if err != nil {
		return fmt.Errorf("insert quote: %w", err)
	}
	return nil
}

func insertQuoteLine(ctx context.Context, tx *sql.Tx, quoteID string, line core.QuoteLine, createdAt time.Time) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO quote_lines (id, quote_id, service_agreement_id, description, quantity_min, unit_rate_amount, unit_rate_currency, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, line.ID, quoteID, line.ServiceAgreementID, line.Description, line.QuantityMin, line.UnitRate.Amount, line.UnitRate.Currency, createdAt.UTC().UnixNano())
	if err != nil {
		return fmt.Errorf("insert quote line: %w", err)
	}
	return nil
}

func (s *QuoteStore) linesByQuote(ctx context.Context, quoteID string) ([]core.QuoteLine, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, quote_id, service_agreement_id, description, quantity_min, unit_rate_amount, unit_rate_currency
FROM quote_lines WHERE quote_id = ? ORDER BY created_at, id`, quoteID)
	if err != nil {
		return nil, fmt.Errorf("get quote lines: %w", err)
	}
	defer rows.Close()
	var lines []core.QuoteLine
	for rows.Next() {
		var line core.QuoteLine
		var amount int64
		var currency string
		if err := rows.Scan(&line.ID, &line.QuoteID, &line.ServiceAgreementID, &line.Description, &line.QuantityMin, &amount, &currency); err != nil {
			return nil, fmt.Errorf("scan quote line: %w", err)
		}
		line.UnitRate = core.Money{Amount: amount, Currency: currency}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quote lines: %w", err)
	}
	return lines, nil
}
