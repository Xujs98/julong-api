package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizePageSizeOptions(t *testing.T) {
	normalized, err := NormalizePageSizeOptions(" 20, 10,100 ")
	require.NoError(t, err)
	assert.Equal(t, "20,10,100", normalized)

	for _, value := range []string{"", "0,10", "10,101", "10,10", "10,one"} {
		t.Run(value, func(t *testing.T) {
			_, err := NormalizePageSizeOptions(value)
			assert.Error(t, err)
		})
	}
}

func TestParsePageSizeOptionsFallsBackForInvalidValues(t *testing.T) {
	assert.Equal(t, []int{25, 50}, ParsePageSizeOptions("25,50"))
	assert.Equal(t, []int{10, 20, 50, 100}, ParsePageSizeOptions("10,10"))
}
