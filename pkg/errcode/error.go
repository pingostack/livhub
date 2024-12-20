package errcode

import "fmt"

type Error interface {
	error
	Code() int
	Unwrap() error
}

type errorCode struct {
	code int
	err  error
}

func (e *errorCode) Error() string {
	if e.err != nil {
		return fmt.Sprintf("code: %d(%s), err: %s", e.code, ErrString(e.code), e.err.Error())
	}
	return fmt.Sprintf("code: %d(%s)", e.code, ErrString(e.code))
}

func (e *errorCode) Code() int {
	return e.code
}

func New(code int, err error) Error {
	return &errorCode{
		code: code,
		err:  err,
	}
}

// Common error codes
const (
	// Success
	Success = 0

	// Common errors 1-99
	ErrUnknown = 1
	ErrEOF     = 16
	ErrDecline = 17
	ErrBreak   = 18
	ErrAgain   = 19

	// Stream errors 100-199
	ErrStreamNotFound      = 100
	ErrStreamClosed        = 101
	ErrStreamExists        = 102
	ErrInvalidStreamType   = 103
	ErrSubStreamClosed     = 104
	ErrSubAlreadyExists    = 105
	ErrPublisherNotSet     = 106
	ErrPublisherIsNil      = 107
	ErrPublisherNotMatch   = 108
	ErrPublisherAlreadySet = 109
	ErrSubStreamNotExists  = 110
)

func ErrString(code int) string {
	switch code {
	case Success:
		return "success"
	case ErrUnknown:
		return "unknown error"
	case ErrEOF:
		return "EOF"
	case ErrDecline:
		return "decline"
	case ErrBreak:
		return "break"
	case ErrAgain:
		return "again"
	case ErrStreamNotFound:
		return "stream not found"
	case ErrStreamClosed:
		return "stream closed"
	case ErrStreamExists:
		return "stream exists"
	case ErrInvalidStreamType:
		return "invalid stream type"
	case ErrSubStreamClosed:
		return "sub stream closed"
	case ErrSubAlreadyExists:
		return "sub already exists"
	case ErrPublisherNotSet:
		return "publisher not set"
	case ErrPublisherIsNil:
		return "publisher is nil"
	case ErrPublisherNotMatch:
		return "publisher not match"
	case ErrPublisherAlreadySet:
		return "publisher already set"
	case ErrSubStreamNotExists:
		return "sub stream not exists"

	default:
		return "unknown error"
	}
}

func ConvError(err error) Error {
	if e, ok := err.(Error); ok {
		return e
	}
	return New(ErrUnknown, err)
}

func Is(err error, code int) bool {
	if err == nil {
		return false
	}

	// unwrap error
	for {
		if e, ok := err.(Error); ok {
			if e.Code() == code {
				return true
			}
			err = e.Unwrap()
			if err == nil {
				break
			}
		} else {
			break
		}
	}

	if e, ok := err.(Error); ok {
		return e.Code() == code
	}

	return false
}

func IsAny(err error, codes ...int) bool {
	for _, code := range codes {
		if Is(err, code) {
			return true
		}
	}
	return false
}

func Unwrap(err error) error {
	if e, ok := err.(Error); ok {
		return e.Unwrap()
	}
	return err
}

func (e *errorCode) Unwrap() error {
	return e.err
}
