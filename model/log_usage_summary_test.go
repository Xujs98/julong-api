package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSumUserUsedTokenBetweenUsesConsumeLogsAndHalfOpenRange(t *testing.T) {
	truncateTables(t)
	logs := []Log{
		{UserId: 1, Type: LogTypeConsume, CreatedAt: 1000, PromptTokens: 10, CompletionTokens: 5},
		{UserId: 1, Type: LogTypeConsume, CreatedAt: 1999, PromptTokens: 20, CompletionTokens: 10},
		{UserId: 1, Type: LogTypeConsume, CreatedAt: 2000, PromptTokens: 100, CompletionTokens: 100},
		{UserId: 1, Type: LogTypeRefund, CreatedAt: 1500, PromptTokens: 50, CompletionTokens: 50},
		{UserId: 2, Type: LogTypeConsume, CreatedAt: 1500, PromptTokens: 50, CompletionTokens: 50},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	tokens, err := SumUserUsedTokenBetween(1, 1000, 2000)
	require.NoError(t, err)
	require.Equal(t, int64(45), tokens)
}
