package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Carlos0934/billar/internal/app"
)

func parseCustomerListFlags(args []string) (app.ListQuery, Format, error) {
	flags := flag.NewFlagSet("customer list", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	var (
		search     string
		sortValue  string
		page       int
		pageSize   int
		formatText string
	)

	flags.StringVar(&search, "search", "", "search by customer name")
	flags.StringVar(&sortValue, "sort", "", "sort field and direction")
	flags.IntVar(&page, "page", 0, "page number")
	flags.IntVar(&pageSize, "page-size", 0, "page size")
	flags.StringVar(&formatText, "format", string(FormatText), "output format")

	if err := flags.Parse(args); err != nil {
		return app.ListQuery{}, "", fmt.Errorf("%s: %w", commandUsage, err)
	}
	if flags.NArg() != 0 {
		return app.ListQuery{}, "", errors.New(commandUsage)
	}

	sortField, sortDir := parseSortValue(sortValue)
	format, err := ParseFormat(formatText)
	if err != nil {
		return app.ListQuery{}, "", err
	}

	return app.ListQuery{
		Page:      page,
		PageSize:  pageSize,
		Search:    search,
		SortField: sortField,
		SortDir:   sortDir,
	}, format, nil
}

func parseSortValue(value string) (string, string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", ""
	}

	if strings.HasPrefix(value, "-") {
		return strings.TrimSpace(strings.TrimPrefix(value, "-")), "desc"
	}

	field, dir, found := strings.Cut(value, ":")
	if !found {
		return strings.TrimSpace(value), ""
	}

	return strings.TrimSpace(field), strings.TrimSpace(dir)
}

func writeCustomerListText(out io.Writer, result app.ListResult[app.CustomerDTO], colorEnabled bool) error {
	view := newTextView(out, colorEnabled)
	view.Title("Billar Customers").Divider("───────────────").Field("Page", strconv.Itoa(result.Page)).Field("Page size", strconv.Itoa(result.PageSize)).Field("Total", strconv.Itoa(result.Total))

	if len(result.Items) == 0 {
		view.Line("No customers found")
		_, err := io.WriteString(out, view.Build())
		return err
	}

	view.Line("")
	for i, customer := range result.Items {
		if i > 0 {
			view.Line("")
		}
		view.Line(fmt.Sprintf("%d. %s", i+1, customer.LegalName))
		if customer.TradeName != "" && customer.TradeName != customer.LegalName {
			view.Line("   Trade name: " + customer.TradeName)
		}
		view.Line("   Type: " + customer.Type)
		view.Line("   Status: " + customer.Status)
		if customer.Email != "" {
			view.Line("   Email: " + customer.Email)
		}
		if customer.DefaultCurrency != "" {
			view.Line("   Default currency: " + customer.DefaultCurrency)
		}
		if customer.CreatedAt != "" {
			view.Line("   Created at: " + customer.CreatedAt)
		}
		if customer.UpdatedAt != "" {
			view.Line("   Updated at: " + customer.UpdatedAt)
		}
	}

	_, err := io.WriteString(out, view.Build())
	return err
}
