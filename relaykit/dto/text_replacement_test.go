package dto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTextReplacementConfigValidateAndApply(t *testing.T) {
	config := TextReplacementConfig{
		Request:  []TextReplacementRule{{From: "source", To: "upstream"}},
		Response: []TextReplacementRule{{From: "upstream", To: "source"}},
	}
	require.NoError(t, config.Validate())
	require.Equal(t, "upstream value", config.Apply("source value", false))
	require.Equal(t, "source value", config.Apply("upstream value", true))
}

func TestTextReplacementConfigRejectsEmptyOrOversizedRules(t *testing.T) {
	require.Error(t, (TextReplacementConfig{Request: []TextReplacementRule{{From: "   "}}}).Validate())
	require.Error(t, (TextReplacementConfig{Response: []TextReplacementRule{{From: strings.Repeat("x", MaxTextReplacementPart+1)}}}).Validate())
	require.Error(t, (TextReplacementConfig{Request: make([]TextReplacementRule, MaxTextReplacementRules+1)}).Validate())
}
