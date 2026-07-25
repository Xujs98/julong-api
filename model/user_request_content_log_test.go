package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserRequestContentLogsRetainNewestPerUser(t *testing.T) {
	truncateTables(t)
	user := User{Username: "request-content-retention", Password: "password"}
	require.NoError(t, DB.Create(&user).Error)

	for i := 0; i < MaxUserRequestContentLogs+2; i++ {
		require.NoError(t, CreateUserRequestContentLog(&UserRequestContentLog{
			UserId:       user.Id,
			RequestId:    fmt.Sprintf("request-%02d", i),
			CreatedAt:    int64(i + 1),
			Status:       UserRequestContentStatusPending,
			OriginalSize: 100,
		}))
	}

	logs, err := GetUserRequestContentLogs(user.Id)
	require.NoError(t, err)
	require.Len(t, logs, MaxUserRequestContentLogs)
	assert.Equal(t, "request-51", logs[0].RequestId)
	assert.Equal(t, "request-02", logs[len(logs)-1].RequestId)

	_, err = GetUserRequestContentLog(user.Id+1, logs[0].Id)
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestFinishUserRequestContentLogPersistsOutcome(t *testing.T) {
	truncateTables(t)
	user := User{Username: "request-content-status", Password: "password"}
	require.NoError(t, DB.Create(&user).Error)
	require.NoError(t, CreateUserRequestContentLog(&UserRequestContentLog{
		UserId: user.Id, RequestId: "request-status", Status: UserRequestContentStatusPending,
	}))

	require.NoError(t, FinishUserRequestContentLog("request-status", UserRequestContentStatusError, "upstream failed"))
	logs, err := GetUserRequestContentLogs(user.Id)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, UserRequestContentStatusError, logs[0].Status)
	assert.Equal(t, "upstream failed", logs[0].ErrorMessage)
}
