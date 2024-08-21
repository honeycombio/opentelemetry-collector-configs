package reduceprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEmptyGroupByReturnsError(t *testing.T) {
	cfg := &Config{
		GroupBy: []string{},
	}
	err := cfg.Validate()
	require.Error(t, err)
	require.Equal(t, "group_by must contain at least one attribute name", err.Error())
}
