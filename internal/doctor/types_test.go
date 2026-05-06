package doctor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxSeverity_Empty(t *testing.T) {
	r := &Result{}
	assert.Equal(t, SeverityOK, r.MaxSeverity())
}

func TestMaxSeverity_Mix(t *testing.T) {
	r := &Result{Checks: []Check{
		{Severity: SeverityOK},
		{Severity: SeverityWarn},
		{Severity: SeverityOK},
	}}
	assert.Equal(t, SeverityWarn, r.MaxSeverity())
}

func TestMaxSeverity_FailWins(t *testing.T) {
	r := &Result{Checks: []Check{
		{Severity: SeverityOK},
		{Severity: SeverityWarn},
		{Severity: SeverityFail},
		{Severity: SeverityWarn},
	}}
	assert.Equal(t, SeverityFail, r.MaxSeverity())
}

func TestCounts(t *testing.T) {
	r := &Result{Checks: []Check{
		{Severity: SeverityOK},
		{Severity: SeverityOK},
		{Severity: SeverityWarn},
		{Severity: SeverityFail},
	}}
	ok, warn, fail := r.Counts()
	assert.Equal(t, 2, ok)
	assert.Equal(t, 1, warn)
	assert.Equal(t, 1, fail)
}
