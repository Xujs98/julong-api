package controller

import (
	"net/http"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	pluginruntime "github.com/QuantumNous/new-api/pkg/jsplugin"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskPluginImageProtocolPreservesJulongImageLogs(t *testing.T) {
	for _, tc := range []struct {
		name    string
		enabled bool
		taskID  string
	}{
		{name: "synchronous logging", enabled: true},
		{name: "existing asynchronous task", taskID: "img_plugin_async"},
		{name: "logging disabled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := setupModelListControllerTestDB(t)
			require.NoError(t, db.AutoMigrate(&model.Token{}, &model.ImageGenerationLog{}))
			require.NoError(t, db.Create(&model.User{Id: 71, Username: "plugin-image-user", Status: common.UserStatusEnabled}).Error)
			require.NoError(t, db.Create(&model.Token{Id: 81, UserId: 71, Name: "plugin-image-token"}).Error)
			previousEnabled := common.ImageGenerationLogEnabled
			common.ImageGenerationLogEnabled = tc.enabled
			t.Cleanup(func() { common.ImageGenerationLogEnabled = previousEnabled })
			if tc.taskID != "" {
				require.NoError(t, db.Create(&model.ImageGenerationLog{
					UserId: 71, TaskId: tc.taskID, Status: model.ImageGenerationStatusProcessing,
				}).Error)
			}

			c, recorder := newImageProtocolTestContext("")
			c.Set(common.RequestIdKey, "req_plugin_julong_image")
			c.Set(service.ImageGenerationTaskContextKey, tc.taskID)
			requestValue, exists := c.Get(pluginruntime.ContextKeyProtocolRequest)
			require.True(t, exists)
			request := requestValue.(pluginruntime.ProtocolRequestContext)
			value := request.Body.(map[string]any)["value"].(map[string]any)
			value["size"] = "1024x1024"
			value["quality"] = "high"
			deps := pluginProtocolTestDeps()
			deps.submit = func(_ *gin.Context, info *relaycommon.RelayInfo) (*taskSubmissionOutcome, *dto.TaskError) {
				info.ChannelMeta = &relaycommon.ChannelMeta{ChannelId: 3}
				outcome := imageProtocolTestOutcome(info, model.TaskStatusSuccess, "https://cdn.example/julong.png")
				outcome.Task.Quota = 123
				return outcome, nil
			}
			serveTaskPluginImageProtocol(c, imageProtocolTestEndpoint(t), deps)
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			var logs []model.ImageGenerationLog
			require.NoError(t, db.Find(&logs).Error)
			if !tc.enabled && tc.taskID == "" {
				assert.Empty(t, logs)
				return
			}
			require.Len(t, logs, 1, "plugin results update the existing async log instead of creating a duplicate")
			log := logs[0]
			assert.Equal(t, tc.taskID, log.TaskId)
			assert.Equal(t, "image-model", log.ModelName)
			assert.Equal(t, "plugin-image-token", log.TokenName)
			assert.Equal(t, "a cat", log.Prompt)
			assert.Equal(t, "1024x1024", log.Size)
			assert.Equal(t, "high", log.Quality)
			assert.Equal(t, 123, log.Quota)
			assert.Equal(t, 1, log.ImageCount)
			assert.Equal(t, 3, log.ChannelId)
			if tc.taskID != "" {
				assert.Equal(t, model.ImageGenerationStatusSuccess, log.Status)
			}
			refs, err := log.ImageRefs()
			require.NoError(t, err)
			require.Len(t, refs, 1)
			assert.Equal(t, "remote", refs[0].Type)
			assert.Equal(t, "https://cdn.example/julong.png", refs[0].Value)
		})
	}
}
