package daemon

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"syscall"
)

// TransportErrorKind names why a request to a daemon socket failed at the
// transport level. See [ClassifyTransportError] for how an error maps onto
// it and [TransportErrorKind.Unavailable] for the "daemon did not answer"
// verdict.
type TransportErrorKind int

// The kinds a transport failure can have.
const (
	// TransportErrorNone: there was no error.
	TransportErrorNone TransportErrorKind = iota

	// TransportErrorCanceled: the caller's own context was done (canceled,
	// or past its own deadline), so the failure is the caller's and says
	// nothing about the daemon.
	TransportErrorCanceled

	// TransportErrorConnectionRefused: nothing accepts on the socket
	// (ECONNREFUSED), e.g. a socket file left behind by a dead daemon.
	TransportErrorConnectionRefused

	// TransportErrorSocketMissing: there is no socket file at that path
	// (ENOENT). The daemon never started, or already removed its socket.
	TransportErrorSocketMissing

	// TransportErrorReset: the connection broke during the request
	// (ECONNRESET, EPIPE, a closed connection, or net/http's "server closed
	// idle connection"). The daemon went away mid-request.
	TransportErrorReset

	// TransportErrorEOF: the daemon closed the connection without sending a
	// response (io.EOF, io.ErrUnexpectedEOF).
	TransportErrorEOF

	// TransportErrorTimeout: the caller's context is alive but the request
	// hit a deadline: the HTTP client's own timeout, or a dial that timed
	// out. The daemon accepted the connection and never answered, or nobody
	// accepted in time.
	TransportErrorTimeout

	// TransportErrorOther: not a recognized transport failure (a bad request
	// URL, a TLS error, an application error). Nothing here says the daemon
	// is down.
	TransportErrorOther
)

// serverClosedIdleFragment is the message of net/http's unexported
// errServerClosedIdle. The sentinel cannot be matched with errors.Is, so its
// text is matched instead; TestGoStillUsesServerClosedIdleConnString pins
// that Go still carries it.
const serverClosedIdleFragment = "server closed idle connection"

// String returns the kind's lower-case name, e.g. "connection refused", for
// log lines and metric labels.
func (kind TransportErrorKind) String() string {
	switch kind {
	case TransportErrorNone:
		return "none"
	case TransportErrorCanceled:
		return "canceled"
	case TransportErrorConnectionRefused:
		return "connection refused"
	case TransportErrorSocketMissing:
		return "socket missing"
	case TransportErrorReset:
		return "reset"
	case TransportErrorEOF:
		return "eof"
	case TransportErrorTimeout:
		return "timeout"
	case TransportErrorOther:
		return "other"
	default:
		return "unknown"
	}
}

// Unavailable reports whether the kind means the daemon could not serve the
// request: connection refused, socket missing, reset, EOF, or timeout.
// Canceled and Other are excluded because they carry no verdict about the
// daemon.
func (kind TransportErrorKind) Unavailable() bool {
	switch kind {
	case TransportErrorConnectionRefused,
		TransportErrorSocketMissing,
		TransportErrorReset,
		TransportErrorEOF,
		TransportErrorTimeout:
		return true
	case TransportErrorNone, TransportErrorCanceled, TransportErrorOther:
		return false
	default:
		return false
	}
}

// ClassifyTransportError names why a request to a daemon socket failed at
// the transport level. err is the error an HTTP client returned; ctx is the
// context the request was issued with.
//
// A done ctx decides first: the caller canceled, or its own deadline
// expired, so the kind is TransportErrorCanceled whatever the error looks
// like. Only a live ctx turns a deadline into TransportErrorTimeout, which
// means the client's own timeout fired while the caller was still waiting.
//
// The classification never alters err. Callers decide what a kind means for
// them; for "is the daemon down?" use [TransportErrorKind.Unavailable].
func ClassifyTransportError(ctx context.Context, err error) TransportErrorKind {
	switch {
	case err == nil:
		return TransportErrorNone
	case ctx.Err() != nil, errors.Is(err, context.Canceled):
		return TransportErrorCanceled
	case errors.Is(err, syscall.ECONNREFUSED):
		return TransportErrorConnectionRefused
	case errors.Is(err, syscall.ENOENT):
		return TransportErrorSocketMissing
	case isReset(err):
		return TransportErrorReset
	case errors.Is(err, io.EOF), errors.Is(err, io.ErrUnexpectedEOF):
		return TransportErrorEOF
	case isDeadline(err):
		return TransportErrorTimeout
	default:
		return TransportErrorOther
	}
}

// isReset reports whether err is a connection that broke under the request.
func isReset(err error) bool {
	return errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.EPIPE) ||
		errors.Is(err, net.ErrClosed) ||
		strings.Contains(err.Error(), serverClosedIdleFragment)
}

// isDeadline reports whether err is a timeout: context.DeadlineExceeded
// (net/http's Client.Timeout error matches it), ETIMEDOUT, or any net.Error
// that reports Timeout.
func isDeadline(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	netErr, ok := errors.AsType[net.Error](err)

	return ok && netErr.Timeout()
}
