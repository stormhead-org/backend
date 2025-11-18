package lib

import (
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// HandleError converts a standard error into a gRPC status error.
// It maps specific, known errors to appropriate gRPC status codes.
func HandleError(err error) error {
	if err == nil {
		return nil
	}

	// Default to internal error unless a specific mapping is found.
	code := codes.Internal
	message := "An unexpected error occurred."

	if errors.Is(err, gorm.ErrRecordNotFound) {
		code = codes.NotFound
		message = "The requested resource was not found."
	}

	// Here you could add more specific error checks, for example:
	// var validationErr *MyValidationError
	// if errors.As(err, &validationErr) {
	// 	code = codes.InvalidArgument
	// 	message = validationErr.Error()
	// }

	return status.Errorf(code, message)
}

// NotFoundError returns a gRPC NotFound error.
func NotFoundError(message string) error {
	if message == "" {
		message = "The requested resource was not found."
	}
	return status.Errorf(codes.NotFound, message)
}

// InternalError returns a gRPC Internal error.
func InternalError() error {
	return status.Errorf(codes.Internal, "An unexpected internal error occurred.")
}

// InvalidArgumentError returns a gRPC InvalidArgument error.
func InvalidArgumentError(message string) error {
	return status.Errorf(codes.InvalidArgument, message)
}

// PermissionDeniedError returns a gRPC PermissionDenied error.
func PermissionDeniedError(message string) error {
	if message == "" {
		message = "You do not have permission to perform this action."
	}
	return status.Errorf(codes.PermissionDenied, message)
}
