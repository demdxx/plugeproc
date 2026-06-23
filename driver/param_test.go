package driver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParam(t *testing.T) {
	p := &Param{
		Name:  "test",
		Type:  TypeString,
		Value: "test-value",
	}

	assert.Equal(t, "{{test}}", p.MacroName())

	val, err := p.ValueStr()
	assert.NoError(t, err)
	assert.Equal(t, "test-value", val)
}
