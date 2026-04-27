package upstream

// DisbursementAPIResponse is the raw JSON returned by the admin service
// GET /v0.1/disbursal-requests/fetch?application_id={appID}
type DisbursementAPIResponse struct {
	Success    bool                 `json:"success"`
	StatusCode int                  `json:"status_code"`
	Message    string               `json:"message"`
	Data       []DisbursementRecord `json:"data"`
}

// DisbursementRecord is one entry in the disbursement list.
// disbursement_type can be null in the API response.
type DisbursementRecord struct {
	DisbursalRequestID int            `json:"disbursal_request_id"`
	ApplicationID      int            `json:"application_id"`
	BankDetails        BankDetails    `json:"bank_details"`
	TranchDetails      TranchDetails  `json:"tranch_details"`
	Utr                string         `json:"utr"`
	DisbursalDate      string         `json:"disbursal_date"`
	DisbursementType   *string        `json:"disbursement_type"`
	IsActive           int            `json:"is_active"`
	CreatedOn          string         `json:"created_on"`
	UpdatedOn          string         `json:"updated_on"`
}

type BankDetails struct {
	Label             string  `json:"label"`
	BeneficiaryName   string  `json:"beneficiary_name"`
	BankAccountNumber string  `json:"bank_account_number"`
	BankName          string  `json:"bank_name"`
	IFSC              string  `json:"ifsc"`
	Amount            interface{} `json:"amount"` // API returns as string ("100000.0") or float — inconsistent
	Comment           string  `json:"comment"`
	DisbursementID    string  `json:"disbursement_id"`
}

type TranchDetails struct {
	Label            string `json:"label"`
	TranchLabel      string `json:"tranch_label"`
	EmiCount         string `json:"emi_count"`
	StartDate        string `json:"start_date"`
	EndDate          string `json:"end_date"`
	AdvanceEmiCount  string `json:"advance_emi_count"`
	DefaultAmount    interface{} `json:"default_amount"` // API returns as string or number — inconsistent
	InterestRate     string `json:"interest_rate"`
	Subvention       string `json:"subvention"`
	DisbursedAmount  string `json:"disbursed_amount"`
}

// ApplicationSummaryAPIResponse is the raw JSON returned by the admin service
// GET /v0.1/applications/summary/{appID}/fetch
type ApplicationSummaryAPIResponse struct {
	Success    bool                    `json:"success"`
	StatusCode int                     `json:"status_code"`
	Message    string                  `json:"message"`
	Data       ApplicationSummaryRecord `json:"data"`
}

// ApplicationSummaryRecord holds the fields we need from the application summary response.
type ApplicationSummaryRecord struct {
	CurrentStage              string   `json:"stage_name"`
	ProductName               string   `json:"product_name"`
	ProductTags               []string `json:"product_tags"`
	CurrentApplicationTracker string   `json:"-"` // TODO: wire up once tracker source is decided
}

// PaymentsAPIResponse is the raw JSON returned by the admin service.
// GET /v1/payments/transactions/fetch?application_id={appID}&page=1
type PaymentsAPIResponse struct {
	Success    bool         `json:"success"`
	StatusCode int          `json:"status_code"`
	Message    string       `json:"message"`
	Data       PaymentsData `json:"data"`
}

type PaymentsData struct {
	Results []PaymentRecord `json:"results"`
}

type PaymentRecord struct {
	Source          string                `json:"source"`
	IsSuccess       int                   `json:"is_success"`
	TransactionData PaymentTransactionData `json:"transaction_data"`
}

type PaymentTransactionData struct {
	TransactionID     string  `json:"transaction_id"`
	TransactionAmount float64 `json:"transaction_amount"`
	Date              string  `json:"date"`
}

// RefundsAPIResponse is the raw JSON returned by the admin service.
// GET /v1/refunds/requests/fetch?application_id={appID}&page_num=1
type RefundsAPIResponse struct {
	Success    bool        `json:"success"`
	StatusCode int         `json:"status_code"`
	Message    string      `json:"message"`
	Data       RefundsData `json:"data"`
}

type RefundsData struct {
	Results []RefundRecord `json:"results"`
}

type RefundRecord struct {
	RefundAmount float64 `json:"refund_amount"`
	Status       string  `json:"status"`
	InitiatedOn  string  `json:"initiated_on"`
	SettledOn    string  `json:"settled_on"`
}

// KycStatusAPIResponse is the raw JSON returned by the KYC service-wrapper endpoint.
// POST /v1/service-wrapper/action/kyc-fetch-status
type KycStatusAPIResponse struct {
	Success    bool          `json:"success"`
	StatusCode int           `json:"status_code"`
	Message    string        `json:"message"`
	Data       KycStatusData `json:"data"`
}

// KycStatusData holds the per-type KYC records returned in the response.
// Null entries mean that KYC type has not been initiated.
type KycStatusData struct {
	CkycData *KycRecord `json:"ckyc_data"`
	OkycData *KycRecord `json:"okyc_data"`
	PkycData *KycRecord `json:"pkyc_data"`
	VkycData *KycRecord `json:"vkyc_data"`
}

// KycRecord is the minimal view of any KYC entry — only status is needed for the graph model.
type KycRecord struct {
	Status string `json:"status"`
}

