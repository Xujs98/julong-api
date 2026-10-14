package controller

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestTaskLogDTOSeparatesUserAdminAndRootDetails(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_public",
		Platform: "document-parser",
		PrivateData: model.TaskPrivateData{
			Key:            "channel-secret-canary",
			UpstreamTaskID: "upstream-private",
			NodeName:       "node-a",
			Execution: &model.TaskExecutionSnapshot{
				RequestID:   "request-public",
				RequestPath: "/v1/documents",
				TaskPlugin: &model.TaskPluginSnapshot{
					Key:     "document-parser",
					Name:    "Document Parser",
					Version: "1.2.3",
					Author: &model.TaskPluginAuthorSnapshot{
						Name: "Community Author",
						URL:  "https://plugins.example/author",
					},
					APIVersion: 1,
					Generation: 42,
				},
			},
		},
	}

	userView := tasksToDto([]*model.Task{task}, false, common.RoleCommonUser)[0]
	assert.Empty(t, userView.Platform)
	assert.Nil(t, userView.AdminInfo)
	assert.Nil(t, userView.RootInfo)

	adminView := tasksToDto([]*model.Task{task}, false, common.RoleAdminUser)[0]
	assert.Equal(t, "document-parser", adminView.Platform)
	require.NotNil(t, adminView.AdminInfo)
	require.NotNil(t, adminView.AdminInfo.TaskPlugin)
	assert.Equal(t, "document-parser", adminView.AdminInfo.TaskPlugin.Key)
	assert.Equal(t, "Document Parser", adminView.AdminInfo.TaskPlugin.Name)
	assert.Equal(t, "1.2.3", adminView.AdminInfo.TaskPlugin.Version)
	require.NotNil(t, adminView.AdminInfo.TaskPlugin.Author)
	assert.Equal(t, "Community Author", adminView.AdminInfo.TaskPlugin.Author.Name)
	assert.Equal(t, "https://plugins.example/author", adminView.AdminInfo.TaskPlugin.Author.URL)
	assert.Equal(t, "request-public", adminView.AdminInfo.RequestID)
	assert.Equal(t, "/v1/documents", adminView.AdminInfo.RequestPath)
	assert.Nil(t, adminView.RootInfo)

	rootView := tasksToDto([]*model.Task{task}, false, common.RoleRootUser)[0]
	require.NotNil(t, rootView.AdminInfo)
	require.NotNil(t, rootView.RootInfo)
	require.NotNil(t, rootView.RootInfo.TaskPlugin)
	assert.Equal(t, 1, rootView.RootInfo.TaskPlugin.APIVersion)
	assert.Equal(t, uint64(42), rootView.RootInfo.TaskPlugin.Generation)
	assert.Equal(t, "upstream-private", rootView.RootInfo.UpstreamTaskID)
	assert.Equal(t, "node-a", rootView.RootInfo.NodeName)

	adminJSON, err := common.Marshal(adminView)
	require.NoError(t, err)
	assert.NotContains(t, string(adminJSON), "channel-secret-canary")
	assert.NotContains(t, string(adminJSON), "upstream-private")

	rootJSON, err := common.Marshal(rootView)
	require.NoError(t, err)
	assert.NotContains(t, string(rootJSON), "channel-secret-canary")
	assert.Contains(t, string(rootJSON), "upstream-private")
}

func TestTaskLogDTODoesNotInventHistoricalPluginProvenance(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_without_snapshot",
		Platform: "document-parser",
	}

	adminView := tasksToDto([]*model.Task{task}, false, common.RoleAdminUser)[0]

	assert.Nil(t, adminView.AdminInfo)
	assert.Nil(t, adminView.RootInfo)
}

func TestTaskLogDTOReplacesLegacyVideoURLWithAvailabilityFlag(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_legacy_video",
		Platform:   "jimeng",
		Action:     constant.TaskActionTextToVideo,
		Status:     model.TaskStatusSuccess,
		FailReason: "https://private-upstream.invalid/video.mp4?signature=secret",
	}

	view := tasksToDto([]*model.Task{task}, false, common.RoleCommonUser)[0]
	assert.True(t, view.LegacyVideoAvailable)
	assert.Empty(t, view.ResultURL)
	assert.Empty(t, view.FailReason)
	encoded, err := common.Marshal(view)
	require.NoError(t, err)
	assert.NotContains(t, string(encoded), "private-upstream.invalid")
	assert.NotContains(t, string(encoded), "result_url")
	assert.Contains(t, string(encoded), "legacy_video_available")
}

func TestTaskLogDTOKeepsFailureReasonAndDoesNotMarkPluginTaskLegacy(t *testing.T) {
	failed := &model.Task{
		TaskID:     "task_failed",
		Platform:   "jimeng",
		Action:     constant.TaskActionTextToVideo,
		Status:     model.TaskStatusFailure,
		FailReason: "provider rejected the request",
	}
	failedView := tasksToDto([]*model.Task{failed}, false, common.RoleCommonUser)[0]
	assert.Equal(t, "provider rejected the request", failedView.FailReason)
	assert.False(t, failedView.LegacyVideoAvailable)

	pluginTask := &model.Task{
		TaskID:     "task_plugin_video",
		Platform:   "community-video",
		Action:     constant.TaskActionTextToVideo,
		Status:     model.TaskStatusSuccess,
		FailReason: "https://stale-upstream.invalid/plugin-video.mp4",
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://private-upstream.invalid/plugin-video.mp4",
			Execution: &model.TaskExecutionSnapshot{
				TaskPlugin: &model.TaskPluginSnapshot{Key: "community-video"},
			},
		},
	}
	pluginView := tasksToDto([]*model.Task{pluginTask}, false, common.RoleCommonUser)[0]
	assert.False(t, pluginView.LegacyVideoAvailable)
	assert.Empty(t, pluginView.ResultURL)
	assert.Empty(t, pluginView.FailReason)
}

func TestGetDashboardTaskRequestUsesTaskOwnershipAndRedactsSnapshot(t *testing.T) {
	setupTaskLogDatabase(t)
	task := &model.Task{TaskID: "task_request", UserId: 7}
	require.NoError(t, model.DB.Create(task).Error)
	task.PrivateData.Execution = &model.TaskExecutionSnapshot{
		RequestID:   "req-task-log",
		TokenName:   "customer-token",
		RequestBody: []byte(`{"model":"video","api_key":"secret","prompt":"hello","face":false,"images":["https://media.example/private.png?signature=secret"]}`),
	}
	require.NoError(t, model.DB.Save(task).Error)

	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Set("id", task.UserId)
	c.Set("role", common.RoleCommonUser)
	c.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	c.Request = httptest.NewRequest(http.MethodGet, "/api/task/"+task.TaskID+"/request", nil)
	GetDashboardTaskRequest(c)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotContains(t, recorder.Body.String(), "secret")
	assert.Contains(t, recorder.Body.String(), "hello")
	assert.Contains(t, recorder.Body.String(), "[MEDIA]")
	assert.NotContains(t, recorder.Body.String(), "media.example")
	assert.Contains(t, recorder.Body.String(), `"face":false`)

	foreign, _ := gin.CreateTestContext(httptest.NewRecorder())
	foreign.Set("id", 999)
	foreign.Set("role", common.RoleCommonUser)
	foreign.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
	foreign.Request = httptest.NewRequest(http.MethodGet, "/api/task/"+task.TaskID+"/request", nil)
	GetDashboardTaskRequest(foreign)
	assert.Equal(t, http.StatusNotFound, foreign.Writer.Status())
}

func TestTaskLogDTOIncludesSafeRequestIdentityOnly(t *testing.T) {
	task := &model.Task{
		TaskID:   "task_request_identity",
		Platform: "video",
		PrivateData: model.TaskPrivateData{Execution: &model.TaskExecutionSnapshot{
			RequestID: "req-1", TokenName: "token-display", RequestBody: []byte(`{"secret":"never-list"}`),
		}},
	}
	view := tasksToDto([]*model.Task{task}, false, common.RoleCommonUser)[0]
	assert.Equal(t, "req-1", view.RequestID)
	assert.Equal(t, "token-display", view.TokenName)
	assert.True(t, view.RequestAvailable)
	assert.False(t, strings.Contains(string(mustMarshalTaskDto(t, view)), "never-list"))
}

func mustMarshalTaskDto(t *testing.T, value any) []byte {
	t.Helper()
	encoded, err := common.Marshal(value)
	require.NoError(t, err)
	return encoded
}

func setupTaskLogDatabase(t *testing.T) {
	t.Helper()
	previous := model.DB
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Task{}))
	model.DB = db
	t.Cleanup(func() {
		model.DB = previous
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestDashboardTaskDetailOwnershipAndPlatformPrivacy(t *testing.T) {
	setupTaskLogDatabase(t)
	task := &model.Task{TaskID: "task_detail", UserId: 7, Platform: "suanliai-video", Status: model.TaskStatusFailure, FailReason: "Unsupported image", Data: []byte(`{"duration":8,"result_url":"https://private.example/signed?secret=value","platform":"private-platform"}`)}
	require.NoError(t, model.DB.Create(task).Error)
	for _, tc := range []struct {
		name               string
		user, role, status int
		platform           bool
	}{
		{"owner", 7, common.RoleCommonUser, 200, false},
		{"foreign user", 8, common.RoleCommonUser, 404, false},
		{"admin", 8, common.RoleAdminUser, 200, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(r)
			c.Set("id", tc.user)
			c.Set("role", tc.role)
			c.Params = gin.Params{{Key: "task_id", Value: task.TaskID}}
			c.Request = httptest.NewRequest("GET", "/api/task/"+task.TaskID+"/detail", nil)
			GetDashboardTaskDetail(c)
			assert.Equal(t, tc.status, r.Code)
			assert.Equal(t, "private, no-store", r.Header().Get("Cache-Control"))
			assert.NotContains(t, r.Body.String(), "private.example")
			assert.NotContains(t, r.Body.String(), "private-platform")
			if tc.platform {
				assert.Contains(t, r.Body.String(), `"platform":"suanliai-video"`)
			} else {
				assert.NotContains(t, r.Body.String(), `"platform"`)
			}
			if tc.status == 200 {
				assert.Contains(t, r.Body.String(), "Unsupported image")
				assert.Contains(t, r.Body.String(), `"duration":8`)
			}
		})
	}
}

func TestUserTaskLogKeepsLegacyAudioCapabilityWithoutPlatform(t *testing.T) {
	task := &model.Task{Platform: constant.TaskPlatformSuno, Status: model.TaskStatusSuccess}
	view := tasksToDto([]*model.Task{task}, false, common.RoleCommonUser)[0]
	assert.Empty(t, view.Platform)
	assert.True(t, view.LegacyAudioAvailable)
}
