package handlers

// Error codes
const (
	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeInvalidID      = "INVALID_ID"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	ErrCodeValidationErr  = "VALIDATION_ERROR"
	ErrCodeNotReady       = "NOT_READY"
	ErrCodeFileNotFound   = "FILE_NOT_FOUND"
)

// Error messages used in HTTP handlers
const (
	// Common error messages
	MsgInternalServerError = "internal server error"
	MsgInvalidContractID   = "invalid contract ID"
	MsgContractNotFound    = "contract not found"
	MsgInvalidRequestBody  = "invalid request body"

	// Customer specific messages
	MsgInvalidCustomerID        = "invalid customer ID"
	MsgFailedToRetrieveCustomer = "failed to retrieve customer"
	MsgCustomerNotFound         = "customer not found"

	// Print job specific messages
	MsgInvalidPrintJobID   = "invalid print job ID"
	MsgFailedToRetrieveJob = "failed to retrieve print job"
	MsgPrintJobNotFound    = "print job not found"
	MsgJobNotCompleted     = "job not completed"
	MsgFileNotFound        = "file not found"
)
