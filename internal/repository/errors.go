package repository

// Error format strings for repository operations
const (
	errFmtBeginTx        = "failed to begin transaction: %w"
	errFmtCommitTx       = "failed to commit transaction: %w"
	errFmtUpdateTotalVal = "failed to update total value: %w"
	errFmtRowsAffected   = "failed to get rows affected: %w"
	errFmtCreateContract = "failed to create contract: %w"
	errFmtCreateItem     = "failed to create contract item: %w"
	errFmtGetContract    = "failed to get contract: %w"
	errFmtGetItems       = "failed to get contract items: %w"
	errFmtScanItem       = "failed to scan contract item: %w"
	errFmtUpdateContract = "failed to update contract: %w"
	errFmtDeleteContract = "failed to delete contract: %w"
	errFmtUpdateStatus   = "failed to update contract status: %w"
	errFmtSignContract   = "failed to sign contract: %w"
)
