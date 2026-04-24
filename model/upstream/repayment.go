package upstream

// RepaymentScheduleAPIResponse is the raw JSON returned by the repayment service.
// GET /v1/repayment-schedule/fetch?application_id={appID}
type RepaymentScheduleAPIResponse struct {
	Success    bool                 `json:"success"`
	StatusCode int                  `json:"status_code"`
	Message    string               `json:"message"`
	Data       RepaymentScheduleData `json:"data"`
}

type RepaymentScheduleData struct {
	RepaymentData RepaymentData `json:"repayment_data"`
}

type RepaymentData struct {
	RepaymentSchedule RepaymentSchedule `json:"repayment_schedule"`
}

type RepaymentSchedule struct {
	CombinedData []RepaymentEntry `json:"combined_data"`
}

// RepaymentEntry is one row in combined_data.
// The "Total" row has label "Total" and no repayment_status — it must be excluded from all calculations.
// due_date format: "D-M-YYYY" (e.g. "5-2-2026")
type RepaymentEntry struct {
	Label           string           `json:"label"`
	PrincipalAmount float64          `json:"principal_amount"`
	TotalEmi        float64          `json:"total_emi"`
	DueDate         string           `json:"due_date"`
	RepaymentStatus *RepaymentStatus `json:"repayment_status"`
}

type RepaymentStatus struct {
	Slug string `json:"slug"`
}
