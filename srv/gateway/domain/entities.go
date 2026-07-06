package domain

import "time"

// Event subjects for NATS
const (
	SubjectCompanyProvisioned = "company.provisioned"
	SubjectCompanyFailed      = "company.provision_failed"

	// Deprovision events
	SubjectCompanyDeprovisionRequested = "company.deprovision_requested"
	SubjectCompanyDeprovisioned        = "company.deprovisioned"
	SubjectCompanyDeprovisionFailed    = "company.deprovision_failed"
)

// CompanyProvisionedEvent is published when a tenant database is ready
type CompanyProvisionedEvent struct {
	CompanyID     string    `json:"company_id"`
	Slug          string    `json:"slug"`
	DatabaseName  string    `json:"database_name"`
	OwnerStaffID  string    `json:"owner_staff_id"`
	ProvisionedAt time.Time `json:"provisioned_at"`
}

// CompanyProvisionFailedEvent is published when provisioning fails
type CompanyProvisionFailedEvent struct {
	CompanyID string    `json:"company_id"`
	Slug      string    `json:"slug"`
	Error     string    `json:"error"`
	FailedAt  time.Time `json:"failed_at"`
}

// CompanyDeprovisionRequestedEvent is published when a company deletion is requested
type CompanyDeprovisionRequestedEvent struct {
	CompanyID   string    `json:"company_id"`
	Slug        string    `json:"slug"`
	HardDelete  bool      `json:"hard_delete"` // true = DROP DB + DELETE row
	RequestedAt time.Time `json:"requested_at"`
}

// CompanyDeprovisionedEvent is published when tenant database is dropped
type CompanyDeprovisionedEvent struct {
	CompanyID       string    `json:"company_id"`
	Slug            string    `json:"slug"`
	DatabaseDropped bool      `json:"database_dropped"`
	DeprovisionedAt time.Time `json:"deprovisioned_at"`
}

// CompanyDeprovisionFailedEvent is published when deprovisioning fails
type CompanyDeprovisionFailedEvent struct {
	CompanyID string    `json:"company_id"`
	Slug      string    `json:"slug"`
	Error     string    `json:"error"`
	FailedAt  time.Time `json:"failed_at"`
}
