package errors

import (
	"bytes"
	"fmt"
	"sync"
)

// MultiError type implements the error interface, and contains the
// Errors used to construct it.
type MultiError struct {
	errors []error
	mtx    sync.Mutex
}

// Error returns a concatenated string of the contained errors
func (es *MultiError) Error() string {
	es.mtx.Lock()
	defer es.mtx.Unlock()
	var buf bytes.Buffer

	if len(es.errors) > 1 {
		fmt.Fprintf(&buf, "%d errors: ", len(es.errors))
	}

	for i, err := range es.errors {
		if i != 0 {
			buf.WriteString("; ")
		}
		buf.WriteString(err.Error())
	}

	return buf.String()
}

// Add adds the error to the error list if it is not nil.
func (es *MultiError) Add(err error) {
	es.mtx.Lock()
	defer es.mtx.Unlock()
	if err == nil {
		return
	}
	if merr, ok := err.(*MultiError); ok {
		es.errors = append(es.errors, merr.errors...)
	} else {
		es.errors = append(es.errors, err)
	}
}

// Err returns the error list as an error or nil if it is empty.
func (es *MultiError) Err() error {
	es.mtx.Lock()
	defer es.mtx.Unlock()
	if len(es.errors) == 0 {
		return nil
	}
	return es
}

// Count shows current errors count.
func (es *MultiError) Count() int {
	es.mtx.Lock()
	defer es.mtx.Unlock()
	return len(es.errors)
}
