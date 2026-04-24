package emailverifier

import (
	"errors"
	"net"
	"strings"
)

const (
	ErrTimeout           = "The connection to the mail server has timed out"
	ErrNoSuchHost        = "Mail server does not exist"
	ErrServerUnavailable = "Mail server is unavailable"
	ErrBlocked           = "Blocked by mail server"
)

type LookupError struct {
	Message string
	Details string
}

func (e *LookupError) Error() string {
	return e.Message
}

func parseMXError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := err.Error()

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		if dnsErr.IsTimeout {
			return &LookupError{Message: ErrTimeout, Details: errMsg}
		}
		if dnsErr.IsNotFound || strings.Contains(errMsg, "no such host") {
			return &LookupError{Message: ErrNoSuchHost, Details: errMsg}
		}
		if strings.Contains(errMsg, "server misbehaving") || strings.Contains(errMsg, "server failure") {
			return &LookupError{Message: ErrServerUnavailable, Details: errMsg}
		}
		return &LookupError{Message: ErrBlocked, Details: errMsg}
	}

	return &LookupError{Message: ErrServerUnavailable, Details: errMsg}
}
