package app

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Carlos0934/billar/internal/core"
)

var ErrQuoteNotFound = errors.New("quote not found")

type QuoteStore interface {
	Create(ctx context.Context, quote *core.Quote) error
	GetByID(ctx context.Context, id string) (*core.Quote, error)
	ListByCustomer(ctx context.Context, customerID string, status ...core.QuoteStatus) ([]core.QuoteSummary, error)
	AddLine(ctx context.Context, quoteID string, line core.QuoteLine) error
	Update(ctx context.Context, quote *core.Quote) error
	Delete(ctx context.Context, id string) error
}

type QuoteService struct {
	quotes     QuoteStore
	profiles   CustomerProfileStore
	agreements ServiceAgreementStore
}

func NewQuoteService(quotes QuoteStore, profiles CustomerProfileStore, agreements ServiceAgreementStore) QuoteService {
	return QuoteService{quotes: quotes, profiles: profiles, agreements: agreements}
}

func (s QuoteService) Create(ctx context.Context, cmd CreateQuoteCommand) (QuoteDTO, error) {
	if s.quotes == nil || s.profiles == nil {
		return QuoteDTO{}, errors.New("quote service dependencies are required")
	}
	profile, err := s.profiles.GetByID(ctx, strings.TrimSpace(cmd.CustomerID))
	if err != nil {
		if errors.Is(err, ErrCustomerProfileNotFound) {
			return QuoteDTO{}, fmt.Errorf("create quote: %w", ErrCustomerProfileNotFound)
		}
		return QuoteDTO{}, fmt.Errorf("create quote: get customer profile: %w", err)
	}
	if profile == nil || !profile.CanReceiveInvoices() {
		return QuoteDTO{}, fmt.Errorf("create quote: %w", ErrCustomerProfileInactive)
	}
	if strings.TrimSpace(cmd.Currency) != profile.DefaultCurrency {
		return QuoteDTO{}, fmt.Errorf("create quote: quote currency %q must match customer currency %q", strings.TrimSpace(cmd.Currency), profile.DefaultCurrency)
	}
	quote, err := core.NewQuote(core.QuoteParams{CustomerID: cmd.CustomerID, Currency: cmd.Currency, Notes: cmd.Notes})
	if err != nil {
		return QuoteDTO{}, fmt.Errorf("create quote: %w", err)
	}
	if err := s.quotes.Create(ctx, &quote); err != nil {
		return QuoteDTO{}, fmt.Errorf("create quote: save quote: %w", err)
	}
	return quoteToDTO(quote), nil
}

func (s QuoteService) Get(ctx context.Context, id string) (QuoteDTO, error) {
	quote, err := s.getQuote(ctx, id)
	if err != nil {
		return QuoteDTO{}, fmt.Errorf("get quote: %w", err)
	}
	return quoteToDTO(*quote), nil
}

func (s QuoteService) List(ctx context.Context, cmd ListQuotesCommand) ([]QuoteSummaryDTO, error) {
	if s.quotes == nil {
		return nil, errors.New("quote store is required")
	}
	if strings.TrimSpace(cmd.CustomerID) == "" {
		return nil, errors.New("customer id is required")
	}
	var statuses []core.QuoteStatus
	if strings.TrimSpace(cmd.Status) != "" {
		status := core.QuoteStatus(strings.TrimSpace(cmd.Status))
		if !status.IsValid() {
			return nil, fmt.Errorf("invalid quote status filter: %s", cmd.Status)
		}
		statuses = append(statuses, status)
	}
	summaries, err := s.quotes.ListByCustomer(ctx, strings.TrimSpace(cmd.CustomerID), statuses...)
	if err != nil {
		return nil, fmt.Errorf("list quotes: %w", err)
	}
	dtos := make([]QuoteSummaryDTO, 0, len(summaries))
	for _, summary := range summaries {
		dtos = append(dtos, quoteSummaryToDTO(summary))
	}
	return dtos, nil
}

func (s QuoteService) AddLine(ctx context.Context, cmd AddQuoteLineCommand) (QuoteDTO, error) {
	if s.quotes == nil || s.agreements == nil {
		return QuoteDTO{}, errors.New("quote service dependencies are required")
	}
	quote, err := s.getQuote(ctx, cmd.QuoteID)
	if err != nil {
		return QuoteDTO{}, fmt.Errorf("add quote line: %w", err)
	}
	agreement, err := s.agreements.GetByID(ctx, strings.TrimSpace(cmd.ServiceAgreementID))
	if err != nil {
		if errors.Is(err, ErrServiceAgreementNotFound) {
			return QuoteDTO{}, fmt.Errorf("add quote line: %w", ErrServiceAgreementNotFound)
		}
		return QuoteDTO{}, fmt.Errorf("add quote line: get service agreement: %w", err)
	}
	if agreement == nil || !agreement.Active {
		return QuoteDTO{}, fmt.Errorf("add quote line: %w", ErrInactiveServiceAgreement)
	}
	if agreement.CustomerProfileID != quote.CustomerID {
		return QuoteDTO{}, fmt.Errorf("add quote line: service agreement customer mismatch")
	}
	if agreement.Currency != quote.Currency {
		return QuoteDTO{}, fmt.Errorf("add quote line: agreement currency %q must match quote currency %q", agreement.Currency, quote.Currency)
	}
	rate, err := core.NewMoney(agreement.HourlyRate, agreement.Currency)
	if err != nil {
		return QuoteDTO{}, fmt.Errorf("add quote line: build rate snapshot: %w", err)
	}
	line, err := core.NewQuoteLine(core.QuoteLineParams{QuoteID: quote.ID, ServiceAgreementID: agreement.ID, Description: cmd.Description, QuantityMin: cmd.QuantityMin, UnitRate: rate}, quote.Currency)
	if err != nil {
		return QuoteDTO{}, fmt.Errorf("add quote line: %w", err)
	}
	if err := s.quotes.AddLine(ctx, quote.ID, line); err != nil {
		return QuoteDTO{}, fmt.Errorf("add quote line: save line: %w", err)
	}
	quote.Lines = append(quote.Lines, line)
	return quoteToDTO(*quote), nil
}

func (s QuoteService) Send(ctx context.Context, id string) (QuoteDTO, error) {
	return s.updateLifecycle(ctx, id, (*core.Quote).Send, "send quote")
}

func (s QuoteService) Accept(ctx context.Context, id string) (QuoteDTO, error) {
	return s.updateLifecycle(ctx, id, (*core.Quote).Accept, "accept quote")
}

func (s QuoteService) Reject(ctx context.Context, id string) (QuoteDTO, error) {
	return s.updateLifecycle(ctx, id, (*core.Quote).Reject, "reject quote")
}

func (s QuoteService) Expire(ctx context.Context, id string) (QuoteDTO, error) {
	return s.updateLifecycle(ctx, id, (*core.Quote).Expire, "expire quote")
}

func (s QuoteService) Delete(ctx context.Context, id string) error {
	quote, err := s.getQuote(ctx, id)
	if err != nil {
		return fmt.Errorf("delete quote: %w", err)
	}
	if err := quote.ValidateDelete(); err != nil {
		return fmt.Errorf("delete quote: %w", err)
	}
	if err := s.quotes.Delete(ctx, quote.ID); err != nil {
		return fmt.Errorf("delete quote: %w", err)
	}
	return nil
}

func (s QuoteService) updateLifecycle(ctx context.Context, id string, transition func(*core.Quote) error, action string) (QuoteDTO, error) {
	quote, err := s.getQuote(ctx, id)
	if err != nil {
		return QuoteDTO{}, fmt.Errorf("%s: %w", action, err)
	}
	if err := transition(quote); err != nil {
		return QuoteDTO{}, fmt.Errorf("%s: %w", action, err)
	}
	if err := s.quotes.Update(ctx, quote); err != nil {
		return QuoteDTO{}, fmt.Errorf("%s: update quote: %w", action, err)
	}
	return quoteToDTO(*quote), nil
}

func (s QuoteService) getQuote(ctx context.Context, id string) (*core.Quote, error) {
	if s.quotes == nil {
		return nil, errors.New("quote store is required")
	}
	if strings.TrimSpace(id) == "" {
		return nil, errors.New("quote id is required")
	}
	quote, err := s.quotes.GetByID(ctx, strings.TrimSpace(id))
	if err != nil {
		if errors.Is(err, ErrQuoteNotFound) {
			return nil, ErrQuoteNotFound
		}
		return nil, fmt.Errorf("get quote: %w", err)
	}
	if quote == nil {
		return nil, ErrQuoteNotFound
	}
	return quote, nil
}
