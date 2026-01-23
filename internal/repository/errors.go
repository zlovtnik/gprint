package repository

// Error format strings for repository operations
const (
	errFmtBeginTx        = "failed to begin transaction: %w"
	errFmtCommitTx       = "failed to commit transaction: %w"
	errFmtUpdateTotalVal = "failed to update total value: %w"
	errFmtRowsAffected   = "failed to get rows affected: %w"
)
