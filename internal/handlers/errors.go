package handlers

// Error codes
const (
	ErrCodeInternalError  = "INTERNAL_ERROR"
	ErrCodeInvalidID      = "INVALID_ID"
	ErrCodeNotFound       = "NOT_FOUND"
	ErrCodeUnauthorized   = "UNAUTHORIZED"
	ErrCodeInvalidRequest = "INVALID_REQUEST"
	ErrCodeInvalidJSON    = "INVALID_JSON"
	ErrCodeValidationErr  = "VALIDATION_ERROR"
	ErrCodeNotReady       = "NOT_READY"
	ErrCodeFileNotFound   = "FILE_NOT_FOUND"
)

// Error messages used in HTTP handlers
const (
	// Common error messages
	MsgInternalServerError = "internal server error"
	MsgInvalidContractID   = "invalid contract id"
	MsgContractNotFound    = "contract not found"
	MsgInvalidRequestBody  = "invalid request body"

	// Contract generation messages
	MsgInvalidGeneratedID  = "invalid generated contract id"
	MsgGeneratedNotFound   = "generated contract not found"
	MsgNoGeneratedContract = "no generated contract found"

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
