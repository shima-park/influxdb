package influxdb

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Some error code constant, ideally we want define common platform codes here
// projects on use platform's error, should have their own central place like this.
const (
	EInternal         = "internal error"
	ENotFound         = "not found"
	EConflict         = "conflict" // action cannot be performed
	EInvalid          = "invalid"  // validation failed
	EEmptyValue       = "empty value"
	EUnavailable      = "unavailable"
	EForbidden        = "forbidden"
	EMethodNotAllowed = "method not allowed"
)

// Error is the error struct of platform.
//
// Errors may have error codes, human-readable messages,
// and a logical stack trace.
//
// The Code targets automated handlers so that recovery can occur.
// Msg is used by the system operator to help diagnose and fix the problem.
// Op and Err chain errors together in a logical stack trace to
// further help operators.
//
// To create a simple error,
//     &Error{
//         Code:ENotFound,
//     }
// To show where the error happens, add Op.
//     &Error{
//         Code: ENotFound,
//         Op: "bolt.FindUserByID"
//     }
// To show an error with a unpredictable value, add the value in Msg.
//     &Error{
//        Code: EConflict,
//        Message: fmt.Sprintf("organization with name %s already exist", aName),
//     }
// To show an error wrapped with another error.
//     &Error{
//         Code:EInternal,
//         Err: err,
//     }.
type Error struct {
	Code string
	Msg  string
	Op   string
	Err  error
}

// Error implement the error interface by outputing the Code and Err.
func (e *Error) Error() string {
	var b strings.Builder

	// Print the current operation in our stack, if any.
	if e.Op != "" {
		fmt.Fprintf(&b, "%s: ", e.Op)
	}

	// If wrapping an error, print its Error() message.
	// Otherwise print the error code & message.
	if e.Err != nil {
		b.WriteString(e.Err.Error())
	} else {
		if e.Code != "" {
			fmt.Fprintf(&b, "<%s>", e.Code)
			if e.Msg != "" {
				b.WriteString(" ")
			}
		}
		b.WriteString(e.Msg)
	}
	return b.String()
}

// ErrorCode returns the code of the root error, if available; otherwise returns EINTERNAL.
func ErrorCode(err error) string {
	if err == nil {
		return ""
	}

	e, ok := err.(*Error)
	if !ok {
		return EInternal
	}

	if e == nil {
		return ""
	}

	if e.Code != "" {
		return e.Code
	}

	if e.Err != nil {
		return ErrorCode(e.Err)
	}

	return EInternal
}

// ErrorOp returns the op of the error, if available; otherwise return empty string.
func ErrorOp(err error) string {
	if err == nil {
		return ""
	}

	e, ok := err.(*Error)
	if !ok {
		return ""
	}

	if e == nil {
		return ""
	}

	if e.Op != "" {
		return e.Op
	}

	if e.Err != nil {
		return ErrorOp(e.Err)
	}

	return ""
}

// ErrorMessage returns the human-readable message of the error, if available.
// Otherwise returns a generic error message.
func ErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	e, ok := err.(*Error)
	if !ok {
		return "An internal error has occurred."
	}

	if e == nil {
		return ""
	}

	if e.Msg != "" {
		return e.Msg
	}

	if e.Err != nil {
		return ErrorMessage(e.Err)
	}

	return "An internal error has occurred."
}

// errEncode an JSON encoding helper that is needed to handle the recursive stack of errors.
type errEncode struct {
	Code string      `json:"code"`              // Code is the machine-readable error code.
	Msg  string      `json:"message,omitempty"` // Msg is a human-readable message.
	Op   string      `json:"op,omitempty"`      // Op describes the logical code operation during error.
	Err  interface{} `json:"error,omitempty"`   // Err is a stack of additional errors.
}

// MarshalJSON recursively marshals the stack of Err.
func (e *Error) MarshalJSON() (result []byte, err error) {
	ee := errEncode{
		Code: e.Code,
		Msg:  e.Msg,
		Op:   e.Op,
	}
	if e.Err != nil {
		if _, ok := e.Err.(*Error); ok {
			_, err := e.Err.(*Error).MarshalJSON()
			if err != nil {
				return result, err
			}
			ee.Err = e.Err
		} else {
			ee.Err = e.Err.Error()
		}
	}
	return json.Marshal(ee)
}

// UnmarshalJSON recursively unmarshals the error stack.
func (e *Error) UnmarshalJSON(b []byte) (err error) {
	ee := new(errEncode)
	err = json.Unmarshal(b, ee)
	e.Code = ee.Code
	e.Msg = ee.Msg
	e.Op = ee.Op
	e.Err = decodeInternalError(ee.Err)
	return err
}

func decodeInternalError(target interface{}) error {
	if errStr, ok := target.(string); ok {
		return errors.New(errStr)
	}
	if internalErrMap, ok := target.(map[string]interface{}); ok {
		internalErr := new(Error)
		if code, ok := internalErrMap["code"].(string); ok {
			internalErr.Code = code
		}
		if msg, ok := internalErrMap["message"].(string); ok {
			internalErr.Msg = msg
		}
		if op, ok := internalErrMap["op"].(string); ok {
			internalErr.Op = op
		}
		internalErr.Err = decodeInternalError(internalErrMap["error"])
		return internalErr
	}
	return nil
}
