package graph

import (
	model "data-hub/model/graph"
	"data-hub/model/upstream"
	"fmt"
	"strconv"
	"time"
)

// strPtr converts a string to *string.
// Returns nil for empty strings so GraphQL returns null instead of "".
func strPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// anyToStrPtr converts an interface{} (JSON string or number) to *string.
// Returns nil for nil/empty values so GraphQL returns null.
func anyToStrPtr(v interface{}) *string {
	if v == nil {
		return nil
	}
	switch x := v.(type) {
	case string:
		return strPtr(x)
	case float64:
		s := strconv.FormatFloat(x, 'f', -1, 64)
		return &s
	default:
		s := fmt.Sprintf("%v", x)
		return strPtr(s)
	}
}

// floatPtr converts float64 to *float64. Returns nil for zero values.
func floatPtr(f float64) *float64 {
	if f == 0 {
		return nil
	}
	return &f
}

// computeRepaymentAndEmi derives all 8 repayment summary fields from combined_data.
// It excludes the "Total" aggregation row and treats "paid"/"posted" slugs as settled.
// due_date format from upstream is "D-M-YYYY" (e.g. "5-2-2026").
func computeRepaymentAndEmi(entries []upstream.RepaymentEntry, today time.Time) *model.RepaymentAndEmi {
	paidSlugs := map[string]bool{"paid": true, "posted": true}
	todayDate := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, time.UTC)

	totalEmis := 0
	emisPaid := 0
	outstandingPrincipal := 0.0
	overdueAmount := 0.0
	overdueEmis := 0
	maxDpd := 0
	var nextEmiDate *string
	var nextEmiAmount *float64

	for _, e := range entries {
		if e.Label == "Total" {
			continue
		}
		totalEmis++

		isPaid := e.RepaymentStatus != nil && paidSlugs[e.RepaymentStatus.Slug]
		if isPaid {
			emisPaid++
			continue
		}

		// Unpaid entry
		outstandingPrincipal += e.PrincipalAmount

		dueDate, err := time.Parse("2-1-2006", e.DueDate)
		if err != nil {
			continue
		}
		dueDateOnly := time.Date(dueDate.Year(), dueDate.Month(), dueDate.Day(), 0, 0, 0, 0, time.UTC)

		if dueDateOnly.Before(todayDate) {
			// Overdue
			overdueEmis++
			overdueAmount += e.TotalEmi
			days := int(todayDate.Sub(dueDateOnly).Hours() / 24)
			if days > maxDpd {
				maxDpd = days
			}
		} else if nextEmiDate == nil {
			// Upcoming — first one in order is next EMI
			d := e.DueDate
			amt := e.TotalEmi
			nextEmiDate = &d
			nextEmiAmount = &amt
		}
	}

	return &model.RepaymentAndEmi{
		TotalEmis:            &totalEmis,
		EmisPaid:             &emisPaid,
		NextEmiDate:          nextEmiDate,
		NextEmiAmount:        nextEmiAmount,
		OutstandingPrincipal: &outstandingPrincipal,
		OverdueAmount:        &overdueAmount,
		OverdueEmis:          &overdueEmis,
		Dpd:                  &maxDpd,
	}
}

// mapTranchDetails maps an upstream TranchDetails to the graph model,
// computing subventionFlag and subventionPercentage from the subvention string.
// subventionFlag is true when subvention parses to a value > 0; false otherwise.
// subventionPercentage is set only when subventionFlag is true.
func mapTranchDetails(t upstream.TranchDetails) *model.TranchDetails {
	flag := false
	var pct *float64

	if v, err := strconv.ParseFloat(t.Subvention, 64); err == nil && v > 0 {
		flag = true
		pct = &v
	}

	return &model.TranchDetails{
		Label:                strPtr(t.Label),
		TranchLabel:          strPtr(t.TranchLabel),
		EmiCount:             strPtr(t.EmiCount),
		StartDate:            strPtr(t.StartDate),
		EndDate:              strPtr(t.EndDate),
		AdvanceEmiCount:      strPtr(t.AdvanceEmiCount),
		TranchAmount:         anyToStrPtr(t.DefaultAmount),
		InterestRate:         strPtr(t.InterestRate),
		Subvention:           strPtr(t.Subvention),
		SubventionFlag:       &flag,
		SubventionPercentage: pct,
	}
}