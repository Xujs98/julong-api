package controller

import (
	"encoding/csv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderUsageLogsCSVIncludesBOMAndPreventsFormulaInjection(t *testing.T) {
	payload, err := renderUsageLogsCSV([]*model.Log{{
		Id:        7,
		CreatedAt: 1,
		Type:      model.LogTypeConsume,
		Username:  "=HYPERLINK(\"https://example.com\")",
		Content:   "+SUM(1,1)",
	}})
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(string(payload), "\xEF\xBB\xBF"))

	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(payload), "\xEF\xBB\xBF")))
	rows, err := reader.ReadAll()
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "'=HYPERLINK(\"https://example.com\")", rows[1][3])
	assert.Equal(t, "'+SUM(1,1)", rows[1][17])
	assert.Equal(t, "consume", rows[1][2])
}
