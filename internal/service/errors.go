package service

import "errors"

// Sentinel errors for service operations
var (
	// ErrNotFound indicates the requested resource was not found
	ErrNotFound = errors.New("resource not found")

	// ErrContractNotFound indicates the contract was not found
	ErrContractNotFound = errors.New("contract not found")

	// ErrCustomerNotFound indicates the customer was not found
	ErrCustomerNotFound = errors.New("customer not found")

	// ErrServiceNotFound indicates the service was not found
	ErrServiceNotFound = errors.New("service not found")

	// ErrPrintJobNotFound indicates the print job was not found
	ErrPrintJobNotFound = errors.New("print job not found")

	// ErrContractCannotUpdate indicates the contract cannot be updated due to its status
	ErrContractCannotUpdate = errors.New("contract cannot be updated in current status")

	// ErrInvalidStatusTransition indicates an invalid contract status transition
	ErrInvalidStatusTransition = errors.New("invalid status transition")

	// ErrCannotSign indicates the contract cannot be signed in its current status
	ErrCannotSign = errors.New("contract cannot be signed in current status")

	// ErrCannotAddItem indicates items cannot be added to the contract in its current status
	ErrCannotAddItem = errors.New("cannot add items to contract in current status")

	// ErrCannotDeleteItem indicates items cannot be deleted from the contract in its current status
	ErrCannotDeleteItem = errors.New("cannot delete items from contract in current status")

	// ErrJobNotCompleted indicates the print job is not yet completed
	ErrJobNotCompleted = errors.New("print job is not completed")

	// ErrOutputFileNotFound indicates the output file is missing
	ErrOutputFileNotFound = errors.New("output file not found")

	// ErrFormatNotSupported indicates the requested format is not supported
	ErrFormatNotSupported = errors.New("format not supported")
)

// ContractError wraps a contract-related error with additional context
type ContractError struct {
	Op      string // Operation that failed
	Err     error  // Underlying error
	Message string // User-friendly message
}

func (e *ContractError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return ""
}

func (e *ContractError) Unwrap() error {
	return e.Err
}

// NewContractError creates a new ContractError
func NewContractError(op string, err error, message string) *ContractError {
	return &ContractError{
		Op:      op,
		Err:     err,
		Message: message,
	}
}
