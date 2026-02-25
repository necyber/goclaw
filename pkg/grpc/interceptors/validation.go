package interceptors

import (
	"context"
	"strings"

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// FieldError represents a validation error for a specific field.
type FieldError struct {
	Field   string
	Message string
}

// FieldErrors is a collection of field validation errors.
type FieldErrors []FieldError

func (e FieldErrors) Error() string {
	if len(e) == 0 {
		return "validation failed"
	}
	var sb strings.Builder
	for i, err := range e {
		if i > 0 {
			sb.WriteString("; ")
		}
		if err.Field != "" {
			sb.WriteString(err.Field)
			sb.WriteString(": ")
		}
		sb.WriteString(err.Message)
	}
	return sb.String()
}

// BusinessRuleError represents a business rule validation error.
type BusinessRuleError struct {
	Message string
}

func (e BusinessRuleError) Error() string {
	if e.Message == "" {
		return "business rule violation"
	}
	return e.Message
}

// RequestValidator allows requests to provide their own validation logic.
type RequestValidator interface {
	Validate() error
}

var requestValidator = validator.New()

// ValidationUnaryInterceptor validates request payloads for unary RPCs.
func ValidationUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if err := validateRequest(req); err != nil {
			return nil, mapValidationError(err)
		}
		return handler(ctx, req)
	}
}

// ValidationStreamInterceptor validates request payloads for streaming RPCs.
func ValidationStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		wrapped := &validatingStream{ServerStream: ss}
		return handler(srv, wrapped)
	}
}

type validatingStream struct {
	grpc.ServerStream
}

func (s *validatingStream) RecvMsg(m interface{}) error {
	if err := s.ServerStream.RecvMsg(m); err != nil {
		return err
	}
	if err := validateRequest(m); err != nil {
		return mapValidationError(err)
	}
	return nil
}

func validateRequest(req interface{}) error {
	if req == nil {
		return FieldErrors{{Message: "request is nil"}}
	}
	if v, ok := req.(RequestValidator); ok {
		return v.Validate()
	}
	if err := requestValidator.Struct(req); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			return toFieldErrors(validationErrors)
		}
		return err
	}
	return nil
}

func toFieldErrors(errs validator.ValidationErrors) FieldErrors {
	fields := make(FieldErrors, 0, len(errs))
	for _, fe := range errs {
		fields = append(fields, FieldError{
			Field:   fe.Namespace(),
			Message: fe.Tag(),
		})
	}
	return fields
}

func mapValidationError(err error) error {
	if err == nil {
		return nil
	}
	if _, ok := status.FromError(err); ok {
		return err
	}
	switch err.(type) {
	case BusinessRuleError, *BusinessRuleError:
		return status.Error(codes.FailedPrecondition, err.Error())
	case FieldErrors, *FieldErrors:
		return status.Error(codes.InvalidArgument, err.Error())
	default:
		return status.Error(codes.InvalidArgument, err.Error())
	}
}
