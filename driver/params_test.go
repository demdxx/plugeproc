package driver

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepareMacros(t *testing.T) {
	params := []*Param{
		{Name: "test", Type: TypeString, Value: "test-value"},
	}

	vals, err := Params(params).PrepareMacros(`"`, `\`, "echo", "{{test}}")
	assert.NoError(t, err)
	assert.Equal(t, `echo "test-value"`, strings.Join(vals, " "))
}
