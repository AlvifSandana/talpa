package model

import "time"

type RiskLevel string

const (
	RiskLow    RiskLevel = "low"
	RiskMedium RiskLevel = "medium"
	RiskHigh   RiskLevel = "high"
)

type Rule struct {
	ID           string
	Command      string
	Category     string
	Pattern      string
	RequiresRoot bool
	Risk         RiskLevel
}

type CandidateItem struct {
	ID           string    `json:"id"`
	RuleID       string    `json:"rule_id"`
	Path         string    `json:"path"`
	SizeBytes    int64     `json:"size_bytes"`
	LastModified time.Time `json:"last_modified"`
	Category     string    `json:"category"`
	Risk         RiskLevel `json:"risk"`
	Selected     bool      `json:"selected"`
	RequiresRoot bool      `json:"requires_root"`
	Result       string    `json:"result"`
}

type Summary struct {
	ItemsTotal          int   `json:"items_total"`
	ItemsSelected       int   `json:"items_selected"`
	EstimatedFreedBytes int64 `json:"estimated_freed_bytes"`
	Errors              int   `json:"errors"`
}

type CommandResult struct {
	SchemaVersion string          `json:"schema_version"`
	Command       string          `json:"command"`
	Timestamp     time.Time       `json:"timestamp"`
	DurationMS    int64           `json:"duration_ms"`
	DryRun        bool            `json:"dry_run,omitempty"`
	Summary       Summary         `json:"summary,omitempty"`
	Items         []CandidateItem `json:"items,omitempty"`
	Metrics       any             `json:"metrics,omitempty"`
}

type OperationLogEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	PlanID     string    `json:"plan_id"`
	Command    string    `json:"command"`
	Action     string    `json:"action"`
	Path       string    `json:"path"`
	RuleID     string    `json:"rule_id"`
	Category   string    `json:"category"`
	SizeBytes  int64     `json:"size_bytes"`
	Risk       string    `json:"risk"`
	Result     string    `json:"result"`
	Error      string    `json:"error"`
	DurationMS int64     `json:"duration_ms"`
	DryRun     bool      `json:"dry_run"`
	UserID     int       `json:"user_id"`
}
