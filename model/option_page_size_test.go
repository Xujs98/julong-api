package model

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPageSizeOptionValidationAndNormalization(t *testing.T) {
	require.NoError(t, validateOptionValue(common.PageSizeOptionsOptionKey, "20, 50"))

	normalized, err := normalizeOptionValue(common.PageSizeOptionsOptionKey, "20, 50")
	require.NoError(t, err)
	assert.Equal(t, "20,50", normalized)
	assert.Error(t, validateOptionValue(common.PageSizeOptionsOptionKey, "20,20"))
}

func TestUpdateOptionMapPageSizeOptionsFallsBackForInvalidValues(t *testing.T) {
	originalMap := common.OptionMap
	originalValue := common.PageSizeOptions
	common.OptionMap = map[string]string{}
	t.Cleanup(func() {
		common.OptionMap = originalMap
		common.PageSizeOptions = originalValue
	})

	require.NoError(t, updateOptionMap(common.PageSizeOptionsOptionKey, "25,50"))
	assert.Equal(t, "25,50", common.OptionMap[common.PageSizeOptionsOptionKey])
	assert.Equal(t, "25,50", common.PageSizeOptions)

	require.NoError(t, updateOptionMap(common.PageSizeOptionsOptionKey, "invalid"))
	assert.Equal(t, common.DefaultPageSizeOptions, common.OptionMap[common.PageSizeOptionsOptionKey])
	assert.Equal(t, common.DefaultPageSizeOptions, common.PageSizeOptions)
}
