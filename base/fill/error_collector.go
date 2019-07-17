package fill

import (
	"fmt"
	"strings"
)

// ErrorCollector collects errors, merging them into one, while noting the path where they happen
type ErrorCollector struct {
	errs   []error
	fields []string
}

// NewErrorCollector creates a new ErrorCollector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errs:   make([]error, 0),
		fields: make([]string, 0),
	}
}

// Add adds a non-nil error to the ErrorCollector, noting the path where the error happened
func (c *ErrorCollector) Add(err error) bool {
	if err == nil {
		return false
	}
	if len(c.fields) > 0 {
		c.errs = append(c.errs, fmt.Errorf("at \"%s\": %s", strings.Join(c.fields, "."), err.Error()))
		return true
	}
	c.errs = append(c.errs, err)
	return true
}

// AsSingleError returns the collected errors as a single error, or nil if there are none
func (c *ErrorCollector) AsSingleError() error {
	if len(c.errs) == 0 {
		return nil
	} else if len(c.errs) == 1 {
		return c.errs[0]
	}
	collect := make([]string, 0)
	for _, err := range c.errs {
		collect = append(collect, err.Error())
	}
	return fmt.Errorf("%s", strings.Join(collect, "\n"))
}

// PushField adds a field to the current path
func (c *ErrorCollector) PushField(fieldName string) {
	c.fields = append(c.fields, fieldName)
}

// PopField removes the most recent field from the current path
func (c *ErrorCollector) PopField() {
	c.fields = c.fields[:len(c.fields)-1]
}
