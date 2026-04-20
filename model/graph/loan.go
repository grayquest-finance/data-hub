package model

// LoanData is the top-level GraphQL type for the loanData query.
// It only holds ApplicationID internally — child resolvers use it to fetch data.
// ApplicationID is never exposed to GraphQL consumers.
type LoanData struct {
	ApplicationID string
}

// Disbursement is one item in the disbursements list returned to the consumer.
// All fields are pointers so absent data returns null in GraphQL (not "" or 0).
type Disbursement struct {
	Utr              *string
	DisbursalDate    *string
	DisbursementType *string
	TranchDetails *TranchDetails
}

type TranchDetails struct {
	Label                *string
	TranchLabel          *string
	EmiCount             *string
	StartDate            *string
	EndDate              *string
	AdvanceEmiCount      *string
	DefaultAmount        *string
	InterestRate         *string
	Subvention           *string
	SubventionFlag       *bool
	SubventionPercentage *float64
}

// ApplicationDetails is the GraphQL model for the applicationDetails field.
// All fields are pointers so absent data returns null instead of zero values.
type ApplicationDetails struct {
	CurrentStage              *string
	ProductName               *string
	ProductTags               []*string
	CurrentApplicationTracker *string
}

// RepaymentAndEmi holds computed repayment summary fields derived from the repayment schedule.
type RepaymentAndEmi struct {
	TotalEmis            *int
	EmisPaid             *int
	NextEmiDate          *string
	NextEmiAmount        *float64
	OutstandingPrincipal *float64
	OverdueAmount        *float64
	OverdueEmis          *int
	Dpd                  *int
}

// Payment is one item in the payments list returned to the consumer.
// Status is derived from is_success (1 → SUCCESS, 0 → FAILED).
// Mode is mapped from the source field (PAYMENT-GATEWAY, NACH, etc.).
type Payment struct {
	TransactionID   *string
	Amount          *float64
	TransactionDate *string
	Status          *string
	Mode            *string
}

// Refund is one item in the refunds list returned to the consumer.
type Refund struct {
	RefundAmount *float64
	Status       *string
	InitiatedOn  *string
	SettledOn    *string
}

// Kyc represents a single KYC record in the kyc list.
// KycType is the KYC variant (okyc, ckyc, pkyc, vkyc).
// KycStatus is the current status as returned by the upstream (e.g. PENDING, REJECTED, VERIFIED).
type Kyc struct {
	KycType   *string
	KycStatus *string
}