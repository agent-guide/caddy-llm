package manager

// Status represents the lifecycle state of a Credential entry.
type Status string

const (
	// StatusUnknown means the credential state could not be determined.
	StatusUnknown Status = "unknown"
	// StatusActive indicates the credential is valid and ready for use.
	StatusActive Status = "active"
	// StatusPending indicates the credential is waiting for an external action.
	StatusPending Status = "pending"
	// StatusRefreshing indicates the credential is undergoing a refresh flow.
	StatusRefreshing Status = "refreshing"
	// StatusError indicates the credential is temporarily unavailable due to errors.
	StatusError Status = "error"
	// StatusDisabled marks the credential as intentionally disabled.
	StatusDisabled Status = "disabled"
)
