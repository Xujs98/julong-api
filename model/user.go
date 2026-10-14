package model

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/relaykit/dto"
	"github.com/QuantumNous/new-api/setting/operation_setting"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const UserNameMaxLength = 20

var userSortColumns = map[string]string{
	"id":            "id",
	"username":      "username",
	"quota":         "quota",
	"group":         "group",
	"created_at":    "created_at",
	"last_login_at": "last_login_at",
}

var ErrBindingAlreadyTaken = errors.New("user binding is already taken")
var ErrBindingInvalid = errors.New("user binding is invalid")

// UserBindingUpdates uses pointers so an omitted field keeps its current value
// while an explicit empty string clears an existing binding.
type UserBindingUpdates struct {
	Email      *string `json:"email,omitempty"`
	GitHubId   *string `json:"github_id,omitempty"`
	DiscordId  *string `json:"discord_id,omitempty"`
	OidcId     *string `json:"oidc_id,omitempty"`
	WeChatId   *string `json:"wechat_id,omitempty"`
	TelegramId *string `json:"telegram_id,omitempty"`
	LinuxDOId  *string `json:"linux_do_id,omitempty"`
}

// ApplyUserBindingUpdatesWithTx validates and applies administrator-provided
// binding values within an existing transaction. Empty values intentionally
// clear a binding; omitted pointers leave it unchanged.
func ApplyUserBindingUpdatesWithTx(tx *gorm.DB, userID int, updates *UserBindingUpdates) error {
	if tx == nil || userID <= 0 || updates == nil {
		return nil
	}
	values := map[string]string{}
	if updates.Email != nil {
		values["email"] = NormalizeEmail(*updates.Email)
	}
	if updates.GitHubId != nil {
		values["github_id"] = strings.TrimSpace(*updates.GitHubId)
	}
	if updates.DiscordId != nil {
		values["discord_id"] = strings.TrimSpace(*updates.DiscordId)
	}
	if updates.OidcId != nil {
		values["oidc_id"] = strings.TrimSpace(*updates.OidcId)
	}
	if updates.WeChatId != nil {
		values["wechat_id"] = strings.TrimSpace(*updates.WeChatId)
	}
	if updates.TelegramId != nil {
		values["telegram_id"] = strings.TrimSpace(*updates.TelegramId)
	}
	if updates.LinuxDOId != nil {
		values["linux_do_id"] = strings.TrimSpace(*updates.LinuxDOId)
	}
	for column, value := range values {
		maxLength := 255
		if column == "email" {
			maxLength = 50
		}
		if len(value) > maxLength {
			return ErrBindingInvalid
		}
		if value == "" {
			continue
		}
		if column == "email" {
			if err := ensureEmailAvailableWithTx(tx, value, userID); err != nil {
				if errors.Is(err, ErrEmailAlreadyTaken) {
					return ErrBindingAlreadyTaken
				}
				return err
			}
			continue
		}
		var count int64
		if err := tx.Model(&User{}).Where(column+" = ? AND id <> ?", value, userID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return ErrBindingAlreadyTaken
		}
	}
	if telegram, ok := values["telegram_id"]; ok {
		if err := ReleaseExternalIdentityWithTx(tx, ExternalIdentityProviderTelegram, userID); err != nil {
			return err
		}
		if telegram != "" {
			if err := ClaimExternalIdentityWithTx(tx, ExternalIdentityProviderTelegram, telegram, userID); err != nil {
				return err
			}
		}
	}
	if len(values) == 0 {
		return nil
	}
	return tx.Model(&User{}).Where("id = ?", userID).Updates(values).Error
}

type UserSortOptions struct {
	SortBy    string
	SortOrder string
}

func NewUserSortOptions(sortBy string, sortOrder string) UserSortOptions {
	normalizedSortBy := strings.ToLower(strings.TrimSpace(sortBy))
	normalizedSortOrder := strings.ToLower(strings.TrimSpace(sortOrder))
	if _, ok := userSortColumns[normalizedSortBy]; !ok {
		normalizedSortBy = "id"
		normalizedSortOrder = "desc"
	} else if normalizedSortOrder != "asc" {
		normalizedSortOrder = "desc"
	}

	return UserSortOptions{
		SortBy:    normalizedSortBy,
		SortOrder: normalizedSortOrder,
	}
}

func (options UserSortOptions) Apply(query *gorm.DB) *gorm.DB {
	columnName, ok := userSortColumns[options.SortBy]
	if !ok {
		columnName = "id"
	}
	q := query.Order(clause.OrderByColumn{
		Column: clause.Column{Name: columnName},
		Desc:   options.SortOrder != "asc",
	})
	if columnName != "id" {
		q = q.Order(clause.OrderByColumn{
			Column: clause.Column{Name: "id"},
			Desc:   true,
		})
	}
	return q
}

func resolveUserSortOptions(sortOptions []UserSortOptions) UserSortOptions {
	if len(sortOptions) == 0 {
		return NewUserSortOptions("", "")
	}
	return sortOptions[0]
}

// User if you add sensitive fields, don't forget to clean them in setupLogin function.
// Otherwise, the sensitive information will be saved on local storage in plain text!
type User struct {
	Id                           int                        `json:"id"`
	Username                     string                     `json:"username" gorm:"unique;index" validate:"max=20"`
	Password                     string                     `json:"password" gorm:"not null;" validate:"min=8,max=20"`
	HasPassword                  bool                       `json:"-" gorm:"-:all"`
	OriginalPassword             string                     `json:"original_password" gorm:"-:all"` // this field is only for Password change verification, don't save it to database!
	DisplayName                  string                     `json:"display_name" gorm:"index" validate:"max=20"`
	Role                         int                        `json:"role" gorm:"type:int;default:1"`   // admin, common
	Status                       int                        `json:"status" gorm:"type:int;default:1"` // enabled, disabled
	Email                        string                     `json:"email" gorm:"index" validate:"max=50"`
	GitHubId                     string                     `json:"github_id" gorm:"column:github_id;index"`
	DiscordId                    string                     `json:"discord_id" gorm:"column:discord_id;index"`
	OidcId                       string                     `json:"oidc_id" gorm:"column:oidc_id;index"`
	WeChatId                     string                     `json:"wechat_id" gorm:"column:wechat_id;index"`
	TelegramId                   string                     `json:"telegram_id" gorm:"column:telegram_id;index"`
	VerificationCode             string                     `json:"verification_code" gorm:"-:all"`                         // this field is only for Email verification, don't save it to database!
	AccessToken                  *string                    `json:"-" gorm:"type:char(32);column:access_token;uniqueIndex"` // this token is for system management
	Quota                        int                        `json:"quota" gorm:"type:int;default:0"`
	UsedQuota                    int                        `json:"used_quota" gorm:"type:int;default:0;column:used_quota"` // used quota
	RequestCount                 int                        `json:"request_count" gorm:"type:int;default:0;"`               // request number
	Group                        string                     `json:"group" gorm:"type:varchar(64);default:'default'"`
	GroupRatioAdjustmentEnabled  bool                       `json:"group_ratio_adjustment_enabled" gorm:"type:bool;column:group_ratio_adjustment_enabled"`
	GroupRatioAdjustment         float64                    `json:"group_ratio_adjustment" gorm:"column:group_ratio_adjustment"`
	AffCode                      string                     `json:"aff_code" gorm:"type:varchar(32);column:aff_code;uniqueIndex"`
	AffCount                     int                        `json:"aff_count" gorm:"type:int;default:0;column:aff_count"`
	AffQuota                     int                        `json:"aff_quota" gorm:"type:int;default:0;column:aff_quota"`           // 邀请剩余额度
	AffHistoryQuota              int                        `json:"aff_history_quota" gorm:"type:int;default:0;column:aff_history"` // 邀请历史额度
	InviterId                    int                        `json:"inviter_id" gorm:"type:int;column:inviter_id;index"`
	IsAgent                      bool                       `json:"is_agent" gorm:"type:bool;default:false;column:is_agent"`
	AgentDiscount                int                        `json:"agent_discount" gorm:"type:int;default:100;column:agent_discount"`
	AgentTopUpLink               string                     `json:"agent_topup_link" gorm:"type:varchar(255);column:agent_topup_link" validate:"max=255"`
	AgentId                      int                        `json:"agent_id,omitempty" gorm:"-:all"`
	AgentUsername                string                     `json:"agent_username,omitempty" gorm:"-:all"`
	TagId                        int                        `json:"tag_id" gorm:"column:tag_id;index"`
	Tag                          *UserTag                   `json:"tag,omitempty" gorm:"-:all"`
	RiskTag                      *UserTag                   `json:"risk_tag,omitempty" gorm:"-:all"`
	DeletedAt                    gorm.DeletedAt             `gorm:"index"`
	LinuxDOId                    string                     `json:"linux_do_id" gorm:"column:linux_do_id;index"`
	AccessTokenCreatedAt         *int64                     `json:"-" gorm:"column:access_token_created_at"`
	Setting                      string                     `json:"setting" gorm:"type:text;column:setting"`
	Remark                       string                     `json:"remark,omitempty" gorm:"type:varchar(255)" validate:"max=255"`
	StripeCustomer               string                     `json:"stripe_customer" gorm:"type:varchar(64);column:stripe_customer;index"`
	CreatedAt                    int64                      `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	LastLoginAt                  int64                      `json:"last_login_at" gorm:"default:0;column:last_login_at"`
	LastLoginIP                  string                     `json:"last_login_ip" gorm:"type:varchar(45);default:'';column:last_login_ip"`
	RequestContentLoggingEnabled bool                       `json:"-" gorm:"type:bool;column:request_content_logging_enabled"`
	RiskDetectionEnabled         bool                       `json:"-" gorm:"type:bool;column:risk_detection_enabled"`
	SharedIPUserCount            int64                      `json:"shared_ip_user_count,omitempty" gorm:"-:all"`
	LastLoginIPBlocked           bool                       `json:"last_login_ip_blocked,omitempty" gorm:"-:all"`
	AuthVersion                  int64                      `json:"-" gorm:"type:bigint;not null;default:1;column:auth_version"`
	AdminPermissions             map[string]map[string]bool `json:"admin_permissions,omitempty" gorm:"-:all"`
	BindingUpdates               *UserBindingUpdates        `json:"binding_updates,omitempty" gorm:"-:all"`
}

func (user *User) ToBaseUser() *UserBase {
	cache := &UserBase{
		Id:                           user.Id,
		Group:                        user.Group,
		GroupRatioAdjustmentEnabled:  user.GroupRatioAdjustmentEnabled,
		GroupRatioAdjustment:         user.GroupRatioAdjustment,
		Quota:                        user.Quota,
		Status:                       user.Status,
		Role:                         user.Role,
		Username:                     user.Username,
		Setting:                      user.Setting,
		Email:                        user.Email,
		RequestContentLoggingEnabled: user.RequestContentLoggingEnabled,
		AuthVersion:                  user.AuthVersion,
		CacheSchema:                  userCacheSchemaVersion,
	}
	return cache
}

// Deprecated: 旧版面板访问令牌，仅在升级后的过渡期内使用；删除 users.access_token 列时一并移除。
func (user *User) GetAccessToken() string {
	if user.AccessToken == nil {
		return ""
	}
	return *user.AccessToken
}

// Deprecated: 旧版面板访问令牌，仅在升级后的过渡期内使用；删除 users.access_token 列时一并移除。
func (user *User) SetAccessToken(token string) {
	user.AccessToken = &token
}

// RevokeUserAccessToken returns the generation actually revoked under the row lock.
//
// Deprecated: 旧版面板访问令牌，仅在升级后的过渡期内使用；删除 users.access_token 列时一并移除。
func RevokeUserAccessToken(id int) (string, error) {
	var tokenRef string
	err := DB.Transaction(func(tx *gorm.DB) error {
		var user User
		if err := lockForUpdate(tx).Select("id", "access_token").First(&user, id).Error; err != nil {
			return err
		}
		tokenRef = AccessTokenFingerprint(user.GetAccessToken())
		if tokenRef == "" {
			return nil
		}
		return tx.Model(&User{}).Where("id = ?", id).Updates(map[string]any{"access_token": nil, "access_token_created_at": nil}).Error
	})
	return tokenRef, err
}

func (user *User) GetSetting() dto.UserSetting {
	setting := dto.UserSetting{}
	if user.Setting != "" {
		err := common.Unmarshal([]byte(user.Setting), &setting)
		if err != nil {
			common.SysLog(common.LogText("failed to unmarshal setting: %s", err.Error()))
		}
	}
	return setting
}

func (user *User) SetSetting(setting dto.UserSetting) {
	settingBytes, err := common.Marshal(setting)
	if err != nil {
		common.SysLog(common.LogText("failed to marshal setting: %s", err.Error()))
		return
	}
	user.Setting = string(settingBytes)
}

func UpdateUserSetting(userId int, setting dto.UserSetting) error {
	if userId == 0 {
		return common.NewMessage("ID is empty!")
	}
	settingBytes, err := common.Marshal(setting)
	if err != nil {
		return err
	}
	settingValue := string(settingBytes)
	if err = DB.Model(&User{}).Where("id = ?", userId).Update("setting", settingValue).Error; err != nil {
		return err
	}
	return updateUserSettingCache(userId, settingValue)
}

// userBindColumns 允许通过 UpdateUserBindColumn 更新的第三方账号绑定列白名单。
// 列名只可能来自代码内部的 provider 实现，白名单是防御纵深，不依赖调用方自律。
var userBindColumns = map[string]bool{
	"github_id":   true,
	"discord_id":  true,
	"oidc_id":     true,
	"linux_do_id": true,
	"wechat_id":   true,
}

// UpdateUserBindColumn 第三方账号绑定字段的专用更新。
// 绑定操作必须只写绑定列：若改为“读取完整用户 → 改一个字段 → 整体更新”，
// 读快照期间并发发生的封禁、降权或分组变更会被旧快照覆盖恢复。
// 角色、状态、分组只允许通过各自带锁/CAS 的专用方法修改。
func UpdateUserBindColumn(userId int, column string, value string) error {
	if userId <= 0 {
		return common.NewMessage("ID is empty!")
	}
	if !userBindColumns[column] {
		return fmt.Errorf("invalid user bind column: %s", column)
	}
	return DB.Model(&User{}).Where("id = ?", userId).Update(column, value).Error
}

// 根据用户角色生成默认的边栏配置
func generateDefaultSidebarConfigForRole(userRole int) string {
	defaultConfig := map[string]any{}

	// 聊天区域 - 所有用户都可以访问
	defaultConfig["chat"] = map[string]any{
		"enabled":    true,
		"playground": true,
		"chat":       true,
	}

	// 控制台区域 - 所有用户都可以访问
	defaultConfig["console"] = map[string]any{
		"enabled":    true,
		"detail":     true,
		"token":      true,
		"log":        true,
		"midjourney": true,
		"task":       true,
	}

	// 个人中心区域 - 所有用户都可以访问
	defaultConfig["personal"] = map[string]any{
		"enabled":  true,
		"topup":    true,
		"invoice":  true,
		"personal": true,
	}

	// 管理员区域 - 根据角色决定
	if userRole == common.RoleAdminUser {
		// 管理员可以访问管理员区域，但不能访问系统设置
		defaultConfig["admin"] = map[string]any{
			"enabled":    true,
			"channel":    true,
			"models":     true,
			"redemption": true,
			"user":       true,
			"setting":    false, // 管理员不能访问系统设置
		}
	} else if userRole == common.RoleRootUser {
		// 超级管理员可以访问所有功能
		defaultConfig["admin"] = map[string]any{
			"enabled":    true,
			"channel":    true,
			"models":     true,
			"redemption": true,
			"user":       true,
			"setting":    true,
		}
	}
	// 普通用户不包含admin区域

	// 转换为JSON字符串
	configBytes, err := common.Marshal(defaultConfig)
	if err != nil {
		common.SysLog(common.LogText("failed to generate the default sidebar config: %s", err.Error()))
		return ""
	}

	return string(configBytes)
}

// CheckUserExistOrDeleted check if user exist or deleted, if not exist, return false, nil, if deleted or exist, return true, nil
func CheckUserExistOrDeleted(username string, email string) (bool, error) {
	var user User

	// err := DB.Unscoped().First(&user, "username = ? or email = ?", username, email).Error
	// check email if empty
	var err error
	email = NormalizeEmail(email)
	if email == "" {
		err = DB.Unscoped().First(&user, "username = ?", username).Error
	} else {
		err = DB.Unscoped().First(&user, "username = ? or LOWER(email) = ?", username, email).Error
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// not exist, return false, nil
			return false, nil
		}
		// other error, return false, err
		return false, err
	}
	// exist, return true, nil
	return true, nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func emailQuery(tx *gorm.DB, email string) *gorm.DB {
	if tx == nil {
		tx = DB
	}
	return tx.Unscoped().Model(&User{}).Where("LOWER(email) = ?", NormalizeEmail(email))
}

func CountUsersByEmail(email string) (int64, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return 0, nil
	}
	var count int64
	err := emailQuery(DB, email).Count(&count).Error
	return count, err
}

func IsEmailAvailable(email string, excludeUserID int) (bool, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return true, nil
	}
	query := emailQuery(DB, email)
	if excludeUserID > 0 {
		query = query.Where("id <> ?", excludeUserID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count == 0, nil
}

func EnsureEmailAvailable(email string, excludeUserID int) error {
	available, err := IsEmailAvailable(email, excludeUserID)
	if err != nil {
		return err
	}
	if !available {
		return ErrEmailAlreadyTaken
	}
	return nil
}

// withNormalizedEmailLock serializes concurrent writers that target the same
// normalized email inside tx, so a "check then write" sequence cannot be raced
// by two transactions. It must be called inside an active transaction; the lock
// is scoped to that transaction and released on commit/rollback.
//
//   - PostgreSQL: transaction-level advisory lock keyed by the normalized email.
//   - MySQL (default REPEATABLE READ): a locking read that takes a next-key/gap
//     lock on the email index, blocking concurrent inserts of the same value.
//   - SQLite: no explicit lock; the single-writer model already serializes the
//     write, so a racing second write fails instead of duplicating.
//
// An empty email is allowed to repeat and needs no serialization.
func withNormalizedEmailLock(tx *gorm.DB, email string, fn func(tx *gorm.DB) error) error {
	if err := lockNormalizedEmail(tx, email); err != nil {
		return err
	}
	return fn(tx)
}

func lockNormalizedEmail(tx *gorm.DB, email string) error {
	email = NormalizeEmail(email)
	if email == "" {
		return nil
	}
	switch {
	case common.UsingMainDatabase(common.DatabaseTypePostgreSQL):
		if err := tx.Exec("SELECT pg_advisory_xact_lock(hashtext(?))", email).Error; err != nil {
			return err
		}
	case common.UsingMainDatabase(common.DatabaseTypeMySQL):
		var ids []int
		if err := tx.Raw("SELECT id FROM users WHERE email = ? FOR UPDATE", email).Scan(&ids).Error; err != nil {
			return err
		}
	}
	return nil
}

func GetMaxUserId() int {
	var user User
	DB.Unscoped().Last(&user)
	return user.Id
}

func GetAllUsers(pageInfo *common.PageInfo, sortOptions ...UserSortOptions) (users []*User, total int64, err error) {
	// Start transaction
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Get total count within transaction
	err = tx.Unscoped().Model(&User{}).Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Get paginated users within same transaction
	order := resolveUserSortOptions(sortOptions)
	err = order.Apply(tx.Unscoped()).Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Omit("password", "access_token").Find(&users).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// Commit transaction
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	fillAgentInfo(users)
	fillUserTagInfo(users)
	fillUserRiskTagInfo(users, time.Now().Unix())
	fillUserIPInfo(users)

	return users, total, nil
}

type UserQuotaSummary struct {
	TotalQuota int64 `json:"total_quota"`
	UserCount  int64 `json:"user_count"`
}

func applyUserSearchFilters(query *gorm.DB, keyword string, group string, role *int, status *int, tagId *int, riskGeneratedAt int64) (*gorm.DB, map[int]*UserRiskReport, error) {
	likeCondition := "username LIKE ? OR email LIKE ? OR display_name LIKE ?"
	likeArgs := []any{"%" + keyword + "%", "%" + keyword + "%", "%" + keyword + "%"}

	if keywordInt, err := strconv.Atoi(keyword); err == nil {
		likeCondition = "id = ? OR " + likeCondition
		likeArgs = append([]any{keywordInt}, likeArgs...)
	}

	query = query.Where("("+likeCondition+")", likeArgs...)
	if group != "" {
		query = query.Where(commonGroupCol+" = ?", group)
	}
	if role != nil {
		query = query.Where("role = ?", *role)
	}
	if status != nil {
		if *status == -1 {
			query = query.Where("deleted_at IS NOT NULL")
		} else {
			query = query.Where("deleted_at IS NULL").Where("status = ?", *status)
		}
	}

	var riskReports map[int]*UserRiskReport
	if tagId == nil {
		return query, riskReports, nil
	}
	if riskTag, builtIn := GetBuiltInUserTag(*tagId); builtIn {
		candidateQuery := query.Session(&gorm.Session{})
		if !common.UserRiskDetectionEnabled {
			candidateQuery = candidateQuery.Where("risk_detection_enabled = ?", true)
		}
		var candidateUserIds []int
		if err := candidateQuery.Pluck("id", &candidateUserIds).Error; err != nil {
			return nil, nil, err
		}

		const riskQueryBatchSize = 500
		riskReports = make(map[int]*UserRiskReport, len(candidateUserIds))
		for start := 0; start < len(candidateUserIds); start += riskQueryBatchSize {
			end := min(start+riskQueryBatchSize, len(candidateUserIds))
			batchReports, err := GetUserRiskReports(candidateUserIds[start:end], UserRiskTagWindowDays, riskGeneratedAt)
			if err != nil {
				return nil, nil, err
			}
			for userId, report := range batchReports {
				riskReports[userId] = report
			}
		}

		matchedUserIds := make([]int, 0)
		for _, userId := range candidateUserIds {
			if report := riskReports[userId]; report != nil && report.Level == riskTag.RiskLevel {
				matchedUserIds = append(matchedUserIds, userId)
			}
		}
		return query.Where("id IN ?", matchedUserIds), riskReports, nil
	}

	return query.Where("tag_id = ?", *tagId), riskReports, nil
}

func GetUserQuotaSummary(keyword string, group string, role *int, status *int, tagId *int) (*UserQuotaSummary, error) {
	query, _, err := applyUserSearchFilters(DB.Unscoped().Model(&User{}), keyword, group, role, status, tagId, time.Now().Unix())
	if err != nil {
		return nil, err
	}
	settings, err := GetUserQuotaSummarySettings()
	if err != nil {
		return nil, err
	}
	if len(settings.ExcludedUserIds) > 0 {
		query = query.Where("id NOT IN ?", settings.ExcludedUserIds)
	}

	var summary UserQuotaSummary
	err = query.Select("COALESCE(SUM(quota), 0) AS total_quota, COUNT(*) AS user_count").Scan(&summary).Error
	return &summary, err
}

func SearchUsers(keyword string, group string, role *int, status *int, tagId *int, startIdx int, num int, sortOptions ...UserSortOptions) ([]*User, int64, error) {
	var users []*User
	var total int64
	var err error
	riskGeneratedAt := time.Now().Unix()
	var riskReports map[int]*UserRiskReport

	// 开始事务
	tx := DB.Begin()
	if tx.Error != nil {
		return nil, 0, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	query, riskReports, err := applyUserSearchFilters(tx.Unscoped().Model(&User{}), keyword, group, role, status, tagId, riskGeneratedAt)
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取总数
	err = query.Count(&total).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 获取分页数据
	order := resolveUserSortOptions(sortOptions)
	err = order.Apply(query.Omit("password", "access_token")).Limit(num).Offset(startIdx).Find(&users).Error
	if err != nil {
		tx.Rollback()
		return nil, 0, err
	}

	// 提交事务
	if err = tx.Commit().Error; err != nil {
		return nil, 0, err
	}

	fillAgentInfo(users)
	fillUserTagInfo(users)
	if riskReports != nil {
		fillUserRiskTagsFromReports(users, riskReports)
	} else {
		fillUserRiskTagInfo(users, riskGeneratedAt)
	}
	fillUserIPInfo(users)

	return users, total, nil
}

func GetUserById(id int, selectAll bool) (*User, error) {
	if id == 0 {
		return nil, common.NewMessage("ID is empty!")
	}
	user := User{Id: id}
	var err error = nil
	if selectAll {
		err = DB.First(&user, "id = ?", id).Error
	} else {
		err = DB.Omit("password", "access_token").First(&user, "id = ?", id).Error
	}
	return &user, err
}

// GetSelfUserById exposes password existence without selecting its hash or PAT.
func GetSelfUserById(id int) (*User, error) {
	if id == 0 {
		return nil, common.NewMessage("ID is empty!")
	}
	var profile struct {
		User
		HasPassword bool `gorm:"column:has_password"`
	}
	err := DB.Model(&User{}).Select([]string{
		"id", "username", "display_name", "role", "status", "email",
		"github_id", "discord_id", "oidc_id", "wechat_id", "telegram_id",
		"group", "quota", "used_quota", "request_count", "aff_code", "aff_count",
		"aff_quota", "aff_history", "inviter_id", "linux_do_id", "setting",
		"stripe_customer", "auth_version", "created_at", "last_login_at", "last_login_ip",
		"is_agent", "agent_discount", "agent_topup_link", "group_ratio_adjustment_enabled", "group_ratio_adjustment",
		"request_content_logging_enabled", "risk_detection_enabled",
		"CASE WHEN password <> '' THEN 1 ELSE 0 END AS has_password",
	}).First(&profile, "id = ?", id).Error
	profile.User.HasPassword = profile.HasPassword
	return &profile.User, err
}

func GetAgentByInviteeId(inviteeId int) (*User, error) {
	if inviteeId == 0 {
		return nil, errors.New("id 为空！")
	}
	var user User
	err := DB.Table("users AS invitee").
		Select("agent.*").
		Joins("JOIN users AS agent ON agent.id = invitee.inviter_id").
		Where("invitee.id = ? AND agent.is_agent = ?", inviteeId, true).
		First(&user).Error
	return &user, err
}

func fillAgentInfo(users []*User) {
	if len(users) == 0 {
		return
	}
	agentIds := make([]int, 0)
	seen := map[int]bool{}
	for _, user := range users {
		if user.InviterId > 0 && !seen[user.InviterId] {
			seen[user.InviterId] = true
			agentIds = append(agentIds, user.InviterId)
		}
	}
	if len(agentIds) == 0 {
		return
	}
	var agents []User
	if err := DB.Select("id", "username").Where("id IN ? AND is_agent = ?", agentIds, true).Find(&agents).Error; err != nil {
		return
	}
	agentById := make(map[int]User, len(agents))
	for _, agent := range agents {
		agentById[agent.Id] = agent
	}
	for _, user := range users {
		if agent, ok := agentById[user.InviterId]; ok {
			user.AgentId = agent.Id
			user.AgentUsername = agent.Username
		}
	}
}

func fillUserTagInfo(users []*User) {
	if len(users) == 0 {
		return
	}
	tagIds := make([]int, 0)
	seen := make(map[int]struct{})
	for _, user := range users {
		if user.TagId <= 0 {
			continue
		}
		if _, exists := seen[user.TagId]; exists {
			continue
		}
		seen[user.TagId] = struct{}{}
		tagIds = append(tagIds, user.TagId)
	}
	if len(tagIds) == 0 {
		return
	}
	var tags []UserTag
	if err := DB.Where("id IN ?", tagIds).Find(&tags).Error; err != nil {
		return
	}
	tagsById := make(map[int]UserTag, len(tags))
	for _, tag := range tags {
		tagsById[tag.Id] = tag
	}
	for _, user := range users {
		if tag, exists := tagsById[user.TagId]; exists {
			tagCopy := tag
			user.Tag = &tagCopy
		}
	}
}

func fillUserIPInfo(users []*User) {
	ips := make([]string, 0, len(users))
	seen := make(map[string]struct{}, len(users))
	for _, user := range users {
		if user.LastLoginIP == "" {
			continue
		}
		if _, ok := seen[user.LastLoginIP]; ok {
			continue
		}
		seen[user.LastLoginIP] = struct{}{}
		ips = append(ips, user.LastLoginIP)
	}
	if len(ips) == 0 {
		return
	}
	counts, countErr := GetSharedLastLoginIPCounts(ips)
	blocked, blockedErr := GetBlockedIPSet(ips)
	if countErr != nil && blockedErr != nil {
		return
	}
	for _, user := range users {
		if countErr == nil {
			user.SharedIPUserCount = counts[user.LastLoginIP]
		}
		if blockedErr == nil {
			user.LastLoginIPBlocked = blocked[user.LastLoginIP]
		}
	}
}

func GetUsersByAgent(agentId int, pageInfo *common.PageInfo) (users []*User, total int64, err error) {
	if agentId == 0 {
		return nil, 0, errors.New("无效的代理用户")
	}
	query := DB.Model(&User{}).Where("inviter_id = ?", agentId)
	if err = query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err = query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).
		Select("id", "username", "display_name", "quota", "used_quota", "request_count", "status", "group", "created_at", "last_login_at", "inviter_id").
		Find(&users).Error
	return users, total, err
}

func GetUserIdByAffCode(affCode string) (int, error) {
	if affCode == "" {
		return 0, errors.New("affiliate code is empty")
	}
	var user User
	err := DB.Select("id").First(&user, "aff_code = ?", affCode).Error
	return user.Id, err
}

// DeleteUserById soft-deletes a user and returns how many scoped access tokens
// were deleted with it.
func DeleteUserById(id int) (int64, error) {
	if id == 0 {
		return 0, common.NewMessage("ID is empty!")
	}
	user := User{Id: id}
	return user.Delete()
}

// HardDeleteUserById permanently deletes a user and returns how many scoped
// access tokens were deleted with it.
func HardDeleteUserById(id int) (int64, error) {
	if id == 0 {
		return 0, common.NewMessage("ID is empty!")
	}
	user := User{Id: id}
	return user.HardDelete()
}

func inviteUser(inviterId int) error {
	result := DB.Model(&User{}).Where("id = ?", inviterId).Updates(map[string]any{
		"aff_count":   gorm.Expr("aff_count + ?", 1),
		"aff_quota":   gorm.Expr("aff_quota + ?", common.QuotaForInviter),
		"aff_history": gorm.Expr("aff_history + ?", common.QuotaForInviter),
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (user *User) TransferAffQuotaToQuota(quota int) error {
	// 检查quota是否小于最小额度
	if float64(quota) < common.QuotaPerUnit {
		return common.NewMessage("Minimum transfer quota is {{min}}!", map[string]any{"min": logger.FormatQuota(common.QuotaFromFloat(common.QuotaPerUnit))})
	}

	// 开始数据库事务
	tx := DB.Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback() // 确保在函数退出时事务能回滚

	// 加锁查询用户以确保数据一致性
	err := lockForUpdate(tx).First(user, user.Id).Error
	if err != nil {
		return err
	}

	// 再次检查用户的AffQuota是否足够
	if user.AffQuota < quota {
		return common.NewMessage("Invitation quota is insufficient!")
	}

	// 更新用户额度
	user.AffQuota -= quota
	user.Quota += quota

	// 保存用户状态
	if err := tx.Save(user).Error; err != nil {
		return err
	}

	// 提交事务
	return tx.Commit().Error
}

func (user *User) prepareForInsert(tx *gorm.DB) error {
	user.Email = NormalizeEmail(user.Email)
	if err := ensureEmailAvailableWithTx(tx, user.Email, 0); err != nil {
		return err
	}
	if user.Password == "" {
		return nil
	}
	var err error
	user.Password, err = common.HashAccountPassword(user.Password)
	return err
}

// BindEmailToUser atomically checks email availability and assigns it to the
// user, serializing concurrent binds of the same email so two accounts cannot
// end up sharing one address. The email is normalized before check and store.
func BindEmailToUser(user *User, email string) error {
	email = NormalizeEmail(email)
	if err := DB.Transaction(func(tx *gorm.DB) error {
		return withNormalizedEmailLock(tx, email, func(tx *gorm.DB) error {
			if err := ensureEmailAvailableWithTx(tx, email, user.Id); err != nil {
				return err
			}
			user.Email = email
			return user.UpdateWithTx(tx, false)
		})
	}); err != nil {
		return err
	}
	return updateUserCache(*user)
}

func ensureEmailAvailableWithTx(tx *gorm.DB, email string, excludeUserID int) error {
	email = NormalizeEmail(email)
	if email == "" {
		return nil
	}
	query := emailQuery(tx, email)
	if excludeUserID > 0 {
		query = query.Where("id <> ?", excludeUserID)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrEmailAlreadyTaken
	}
	return nil
}

func (user *User) Insert(inviterId int) error {
	if err := DB.Transaction(func(tx *gorm.DB) error {
		return withNormalizedEmailLock(tx, user.Email, func(tx *gorm.DB) error {
			if err := user.prepareForInsert(tx); err != nil {
				return err
			}
			user.Quota = common.QuotaForNewUser
			user.AffCode = common.GetRandomString(4)

			// 初始化用户设置，包括默认的边栏配置
			if user.Setting == "" {
				defaultSetting := dto.UserSetting{}
				// 这里暂时不设置SidebarModules，因为需要在用户创建后根据角色设置
				user.SetSetting(defaultSetting)
			}

			return tx.Create(user).Error
		})
	}); err != nil {
		return err
	}

	user.finishInsert(inviterId)
	return nil
}

func (user *User) finishInsert(inviterId int) {
	// 用户创建成功后，根据角色初始化边栏配置
	// 需要重新获取用户以确保有正确的ID和Role
	var createdUser User
	if err := DB.Where("username = ?", user.Username).First(&createdUser).Error; err == nil {
		// 生成基于角色的默认边栏配置
		defaultSidebarConfig := generateDefaultSidebarConfigForRole(createdUser.Role)
		if defaultSidebarConfig != "" {
			currentSetting := createdUser.GetSetting()
			currentSetting.SidebarModules = defaultSidebarConfig
			createdUser.SetSetting(currentSetting)
			createdUser.Update(false)
			common.SysLog(common.LogText("initialized the sidebar config for new user %s (role: %d)", createdUser.Username, createdUser.Role))
		}
	}

	if common.QuotaForNewUser > 0 {
		content := fmt.Sprintf("新用户注册赠送 %s", logger.LogQuota(common.QuotaForNewUser))
		RecordLog(user.Id, LogTypeSystem, common.NewMessage(content))
		RecordQuotaIncreaseLog(user.Id, common.QuotaForNewUser, QuotaIncreaseSourceRegistrationBonus, content)
	}
	if inviterId != 0 && operation_setting.IsPaymentComplianceConfirmed() {
		if common.QuotaForInvitee > 0 {
			if err := IncreaseUserQuota(user.Id, common.QuotaForInvitee, true); err == nil {
				content := fmt.Sprintf("使用邀请码赠送 %s", logger.LogQuota(common.QuotaForInvitee))
				RecordLog(user.Id, LogTypeSystem, common.NewMessage(content))
				RecordQuotaIncreaseLog(user.Id, common.QuotaForInvitee, QuotaIncreaseSourceInvitationBonus, content)
			} else {
				common.SysError("failed to add invitee quota: " + err.Error())
			}
		}
		if common.QuotaForInviter > 0 {
			//_ = IncreaseUserQuota(inviterId, common.QuotaForInviter)
			RecordLog(inviterId, LogTypeSystem, common.NewMessage("Bonus for inviting a user {{quota}}", map[string]any{"quota": logger.FormatQuota(common.QuotaForInviter)}))
			_ = inviteUser(inviterId)
		}
	}
}

func (user *User) FinishInsert(inviterId int) {
	user.finishInsert(inviterId)
}

// InsertWithTx inserts a new user within an existing transaction.
// This is used for OAuth registration where user creation and binding need to be atomic.
// Post-creation tasks (sidebar config, logs, inviter rewards) are handled after the transaction commits.
func (user *User) InsertWithTx(tx *gorm.DB, inviterId int) error {
	return withNormalizedEmailLock(tx, user.Email, func(tx *gorm.DB) error {
		if err := user.prepareForInsert(tx); err != nil {
			return err
		}
		user.Quota = common.QuotaForNewUser
		user.AffCode = common.GetRandomString(4)

		// 初始化用户设置
		if user.Setting == "" {
			defaultSetting := dto.UserSetting{}
			user.SetSetting(defaultSetting)
		}

		return tx.Create(user).Error
	})
}

// FinalizeOAuthUserCreation performs post-transaction tasks for OAuth user creation.
// This should be called after the transaction commits successfully.
func (user *User) FinalizeOAuthUserCreation(inviterId int) {
	// 用户创建成功后，根据角色初始化边栏配置
	var createdUser User
	if err := DB.Where("id = ?", user.Id).First(&createdUser).Error; err == nil {
		defaultSidebarConfig := generateDefaultSidebarConfigForRole(createdUser.Role)
		if defaultSidebarConfig != "" {
			currentSetting := createdUser.GetSetting()
			currentSetting.SidebarModules = defaultSidebarConfig
			createdUser.SetSetting(currentSetting)
			createdUser.Update(false)
			common.SysLog(common.LogText("initialized the sidebar config for new user %s (role: %d)", createdUser.Username, createdUser.Role))
		}
	}

	if common.QuotaForNewUser > 0 {
		content := fmt.Sprintf("新用户注册赠送 %s", logger.LogQuota(common.QuotaForNewUser))
		RecordLog(user.Id, LogTypeSystem, common.NewMessage(content))
		RecordQuotaIncreaseLog(user.Id, common.QuotaForNewUser, QuotaIncreaseSourceRegistrationBonus, content)
	}
	if inviterId != 0 && operation_setting.IsPaymentComplianceConfirmed() {
		if common.QuotaForInvitee > 0 {
			if err := IncreaseUserQuota(user.Id, common.QuotaForInvitee, true); err == nil {
				content := fmt.Sprintf("使用邀请码赠送 %s", logger.LogQuota(common.QuotaForInvitee))
				RecordLog(user.Id, LogTypeSystem, common.NewMessage(content))
				RecordQuotaIncreaseLog(user.Id, common.QuotaForInvitee, QuotaIncreaseSourceInvitationBonus, content)
			} else {
				common.SysError("failed to add invitee quota: " + err.Error())
			}
		}
		if common.QuotaForInviter > 0 {
			RecordLog(inviterId, LogTypeSystem, common.NewMessage("Bonus for inviting a user {{quota}}", map[string]any{"quota": logger.FormatQuota(common.QuotaForInviter)}))
			_ = inviteUser(inviterId)
		}
	}
}

func (user *User) Update(updatePassword bool) error {
	var previousAuthVersion int64
	if err := DB.Model(&User{}).Where("id = ?", user.Id).Select("auth_version").Find(&previousAuthVersion).Error; err != nil {
		return err
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		return user.UpdateWithTx(tx, updatePassword)
	}); err != nil {
		return err
	}
	if err := updateUserCache(*user); err != nil {
		return err
	}
	if user.AuthVersion > previousAuthVersion {
		_, err := RevokeAllUserSessions(user.Id, "user_security_changed")
		return err
	}
	return nil
}

func (user *User) UpdateWithTx(tx *gorm.DB, updatePassword bool) error {
	var err error
	if updatePassword {
		user.Password, err = common.HashAccountPassword(user.Password)
		if err != nil {
			return err
		}
	}
	newUser := *user
	current := User{}
	if err = tx.First(&current, user.Id).Error; err != nil {
		return err
	}
	// Updates(struct) ignores zero values. Match that behavior when deciding
	// whether this request actually changes authentication-sensitive state;
	// partial self-profile updates intentionally leave role/status/group empty.
	authChanged := (updatePassword && current.Password != newUser.Password) ||
		(newUser.Role != 0 && current.Role != newUser.Role) ||
		(newUser.Status != 0 && current.Status != newUser.Status) ||
		(newUser.Group != "" && current.Group != newUser.Group)
	if authChanged {
		newUser.AuthVersion, err = IncrementUserAuthVersionWithTx(tx, user.Id)
		if err != nil {
			return err
		}
	}
	if err = tx.Model(&current).Omit(
		"access_token",
		"quota",
		"used_quota",
		"request_count",
		"aff_count",
		"aff_quota",
		"aff_history",
		"auth_version",
	).Updates(newUser).Error; err != nil {
		return err
	}
	return tx.First(user, user.Id).Error
}

func (user *User) Edit(updatePassword bool) error {
	var previousAuthVersion int64
	if err := DB.Model(&User{}).Where("id = ?", user.Id).Select("auth_version").Find(&previousAuthVersion).Error; err != nil {
		return err
	}
	if err := DB.Transaction(func(tx *gorm.DB) error {
		return user.EditWithTx(tx, updatePassword)
	}); err != nil {
		return err
	}
	if err := updateUserCache(*user); err != nil {
		return err
	}
	if user.AuthVersion > previousAuthVersion {
		_, err := RevokeAllUserSessions(user.Id, "user_security_changed")
		return err
	}
	return nil
}

func (user *User) EditWithTx(tx *gorm.DB, updatePassword bool) error {
	var err error
	if updatePassword {
		user.Password, err = common.HashAccountPassword(user.Password)
		if err != nil {
			return err
		}
	}

	newUser := *user
	updates := map[string]interface{}{
		"username":                       newUser.Username,
		"display_name":                   newUser.DisplayName,
		"group":                          newUser.Group,
		"group_ratio_adjustment_enabled": newUser.GroupRatioAdjustmentEnabled,
		"group_ratio_adjustment":         newUser.GroupRatioAdjustment,
		"remark":                         newUser.Remark,
		"is_agent":                       newUser.IsAgent,
		"agent_discount":                 newUser.AgentDiscount,
		"agent_topup_link":               newUser.AgentTopUpLink,
	}
	if updatePassword {
		updates["password"] = newUser.Password
	}

	current := User{}
	if err = tx.First(&current, user.Id).Error; err != nil {
		return err
	}
	authChanged := (updatePassword && current.Password != newUser.Password) || current.Group != newUser.Group
	if authChanged {
		newUser.AuthVersion, err = IncrementUserAuthVersionWithTx(tx, user.Id)
		if err != nil {
			return err
		}
	}
	if err = tx.Model(&current).Updates(updates).Error; err != nil {
		return err
	}
	return tx.First(user, user.Id).Error
}

func (user *User) ClearBinding(bindingType string) error {
	if user.Id == 0 {
		return errors.New("user id is empty")
	}

	bindingColumnMap := map[string]string{
		"email":    "email",
		"github":   "github_id",
		"discord":  "discord_id",
		"oidc":     "oidc_id",
		"wechat":   "wechat_id",
		"telegram": "telegram_id",
		"linuxdo":  "linux_do_id",
	}

	column, ok := bindingColumnMap[bindingType]
	if !ok {
		return errors.New("invalid binding type")
	}

	if err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&User{}).Where("id = ?", user.Id).Update(column, "").Error; err != nil {
			return err
		}
		if bindingType == ExternalIdentityProviderTelegram {
			return ReleaseExternalIdentityWithTx(tx, ExternalIdentityProviderTelegram, user.Id)
		}
		return nil
	}); err != nil {
		return err
	}

	if err := DB.Where("id = ?", user.Id).First(user).Error; err != nil {
		return err
	}

	return updateUserCache(*user)
}

func (user *User) Delete() (int64, error) {
	return user.delete(nil)
}

func DeleteUserForSession(identity AuthSessionIdentity) (int64, error) {
	user := User{Id: identity.UserID}
	return user.delete(&identity)
}

func (user *User) delete(identity *AuthSessionIdentity) (int64, error) {
	if user.Id == 0 {
		return 0, common.NewMessage("ID is empty!")
	}
	var nextAuthVersion int64
	var revokedAccessTokens int64
	if err := DB.Transaction(func(tx *gorm.DB) error {
		if identity != nil {
			if err := ValidateAuthSessionWithTx(tx, *identity); err != nil {
				return err
			}
			var role int
			if err := tx.Model(&User{}).Where("id = ?", user.Id).Select("role").Scan(&role).Error; err != nil {
				return err
			}
			if role == common.RoleRootUser {
				return ErrCannotDeleteRootUser
			}
		}
		var err error
		nextAuthVersion, err = IncrementUserAuthVersionWithTx(tx, user.Id)
		if err != nil {
			return err
		}
		revokedAccessTokens, err = DeleteUserAccessTokensWithTx(tx, user.Id)
		if err != nil {
			return err
		}
		return tx.Delete(user).Error
	}); err != nil {
		return 0, err
	}
	if err := publishCommittedUserAuthVersion(user.Id, nextAuthVersion); err != nil {
		return revokedAccessTokens, err
	}
	if _, err := RevokeAllUserSessions(user.Id, "user_deleted"); err != nil {
		return revokedAccessTokens, err
	}
	return revokedAccessTokens, invalidateUserCache(user.Id)
}

func (user *User) HardDelete() (int64, error) {
	if user.Id == 0 {
		return 0, common.NewMessage("ID is empty!")
	}
	var tokens []Token
	var deletedAuthVersion int64
	var revokedAccessTokens int64
	err := DB.Transaction(func(tx *gorm.DB) error {
		var err error
		deletedAuthVersion, err = IncrementUserAuthVersionWithTx(tx, user.Id)
		if err != nil {
			return err
		}
		revokedAccessTokens, err = DeleteUserAccessTokensWithTx(tx, user.Id)
		if err != nil {
			return err
		}
		if common.RedisEnabled {
			if err := tx.Unscoped().Select("id", commonKeyCol).Where("user_id = ?", user.Id).Find(&tokens).Error; err != nil {
				return err
			}
		}
		if err := deleteUserAuthenticationData(tx, user.Id); err != nil {
			return err
		}
		return tx.Unscoped().Delete(user).Error
	})
	if err != nil {
		return 0, err
	}
	if err := publishCommittedUserAuthVersion(user.Id, deletedAuthVersion); err != nil {
		common.SysError(common.LogText("failed to publish auth tombstone after hard deleting user %d: %v", user.Id, err))
	}
	if err := invalidateTokensCache(tokens); err != nil {
		common.SysError(common.LogText("failed to invalidate token cache after hard deleting user %d: %v", user.Id, err))
	}
	if err := invalidateUserCache(user.Id); err != nil {
		common.SysError(common.LogText("failed to invalidate user cache after hard deleting user %d: %v", user.Id, err))
	}
	return revokedAccessTokens, nil
}

func deleteUserAuthenticationData(tx *gorm.DB, userId int) error {
	if err := releaseAllExternalIdentitiesWithTx(tx, userId); err != nil {
		return err
	}
	for _, authenticationData := range []any{
		&TwoFABackupCode{},
		&TwoFA{},
		&UserSession{},
		&AuthFlow{},
		&PasskeyCredential{},
		&Token{},
		&UserRequestContentLog{},
	} {
		if err := tx.Unscoped().Where("user_id = ?", userId).Delete(authenticationData).Error; err != nil {
			return err
		}
	}
	return deleteUserOAuthBindingsByUserId(tx, userId)
}

// ValidateAndFill check password & user status
func (user *User) ValidateAndFill() (err error) {
	// When querying with struct, GORM will only query with non-zero fields,
	// that means if your field's value is 0, '', false or other zero values,
	// it won't be used to build query conditions
	password := user.Password
	username := strings.TrimSpace(user.Username)
	if username == "" || password == "" {
		return ErrUserEmptyCredentials
	}
	// find by username or email
	err = DB.Where("username = ? OR email = ?", username, username).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrInvalidCredentials
		}
		return fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	if user.Password == "" {
		return ErrInvalidCredentials
	}
	okay := common.ValidatePasswordAndHash(password, user.Password)
	if !okay || user.Status != common.UserStatusEnabled {
		return ErrInvalidCredentials
	}
	return nil
}

func (user *User) FillUserById() error {
	if user.Id == 0 {
		return common.NewMessage("ID is empty!")
	}
	DB.Where(User{Id: user.Id}).First(user)
	return nil
}

func (user *User) FillUserByEmail() error {
	if user.Email == "" {
		return common.NewMessage("Email is empty!")
	}
	DB.Where(User{Email: user.Email}).First(user)
	return nil
}

func (user *User) FillUserByGitHubId() error {
	if user.GitHubId == "" {
		return common.NewMessage("GitHub ID is empty!")
	}
	DB.Where(User{GitHubId: user.GitHubId}).First(user)
	return nil
}

func (user *User) FillUserByDiscordId() error {
	if user.DiscordId == "" {
		return common.NewMessage("Discord ID is empty!")
	}
	DB.Where(User{DiscordId: user.DiscordId}).First(user)
	return nil
}

func (user *User) FillUserByOidcId() error {
	if user.OidcId == "" {
		return common.NewMessage("OIDC ID is empty!")
	}
	DB.Where(User{OidcId: user.OidcId}).First(user)
	return nil
}

func (user *User) FillUserByWeChatId() error {
	if user.WeChatId == "" {
		return common.NewMessage("WeChat ID is empty!")
	}
	DB.Where(User{WeChatId: user.WeChatId}).First(user)
	return nil
}

func (user *User) FillUserByTelegramId() error {
	if user.TelegramId == "" {
		return common.NewMessage("Telegram ID is empty!")
	}
	err := DB.Where(User{TelegramId: user.TelegramId}).First(user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return common.NewMessage("This Telegram account is not linked. Sign in using another method and link it first.")
	}
	return nil
}

func IsEmailAlreadyTaken(email string) bool {
	count, err := CountUsersByEmail(email)
	return err == nil && count > 0
}

func GetUniqueUserByEmail(email string) (*User, error) {
	email = NormalizeEmail(email)
	if email == "" {
		return nil, ErrEmailNotFound
	}
	var users []User
	if err := DB.Where("LOWER(email) = ?", email).Limit(2).Find(&users).Error; err != nil {
		return nil, err
	}
	switch len(users) {
	case 0:
		return nil, ErrEmailNotFound
	case 1:
		return &users[0], nil
	default:
		return nil, ErrEmailAmbiguous
	}
}

func IsWeChatIdAlreadyTaken(wechatId string) bool {
	return DB.Unscoped().Where("wechat_id = ?", wechatId).Find(&User{}).RowsAffected == 1
}

func IsGitHubIdAlreadyTaken(githubId string) bool {
	return DB.Unscoped().Where("github_id = ?", githubId).Find(&User{}).RowsAffected == 1
}

func IsDiscordIdAlreadyTaken(discordId string) bool {
	return DB.Unscoped().Where("discord_id = ?", discordId).Find(&User{}).RowsAffected == 1
}

func IsOidcIdAlreadyTaken(oidcId string) bool {
	return DB.Where("oidc_id = ?", oidcId).Find(&User{}).RowsAffected == 1
}

func IsTelegramIdAlreadyTaken(telegramId string) bool {
	return DB.Unscoped().Where("telegram_id = ?", telegramId).Find(&User{}).RowsAffected == 1
}

func ResetUserPasswordByEmail(email string, password string) error {
	if email == "" || password == "" {
		return common.NewMessage("Email or password is empty!")
	}
	user, err := GetUniqueUserByEmail(email)
	if err != nil {
		return err
	}
	hashedPassword, err := common.HashAccountPassword(password)
	if err != nil {
		return err
	}
	if err = DB.Transaction(func(tx *gorm.DB) error {
		if _, err := IncrementUserAuthVersionWithTx(tx, user.Id); err != nil {
			return err
		}
		return tx.Model(&User{}).Where("id = ?", user.Id).Update("password", hashedPassword).Error
	}); err != nil {
		return err
	}
	if err := PublishUserAuthCache(user.Id); err != nil {
		return err
	}
	_, err = RevokeAllUserSessions(user.Id, "password_reset")
	return err
}

func IsAdmin(userId int) bool {
	if userId == 0 {
		return false
	}
	var user User
	err := DB.Where("id = ?", userId).Select("role").Find(&user).Error
	if err != nil {
		common.SysLog(common.LogText("no such user %s", err.Error()))
		return false
	}
	return user.Role >= common.RoleAdminUser
}

// ValidateAccessToken resolves a legacy plaintext access token. After the
// transition deadline it rejects every value without querying the database.
//
// Deprecated: 旧版面板访问令牌，仅在升级后的过渡期内使用；删除 users.access_token 列时一并移除。
func ValidateAccessToken(token string) (*User, error) {
	if token == "" {
		return nil, nil
	}
	if LegacyAccessTokensRetired(common.GetTimestamp()) {
		return nil, ErrLegacyAccessTokenRetired
	}
	token = strings.Replace(token, "Bearer ", "", 1)
	user := &User{}
	err := DB.Where("access_token = ?", token).First(user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("%w: %v", ErrDatabase, err)
	}
	return user, nil
}

// GetUserQuota gets quota from Redis first, falls back to DB if needed
func GetUserQuota(id int, fromDB bool) (quota int, err error) {
	if !fromDB && common.RedisEnabled {
		return getUserQuotaCache(id)
	}
	err = DB.Model(&User{}).Where("id = ?", id).Select("quota").Find(&quota).Error
	if err != nil {
		return 0, err
	}

	return quota, nil
}

func GetUserUsedQuota(id int) (quota int, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("used_quota").Find(&quota).Error
	return quota, err
}

func GetUserEmail(id int) (email string, err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Select("email").Find(&email).Error
	return email, err
}

// GetUserGroup gets group from Redis first, falls back to DB if needed
func GetUserGroup(id int, fromDB bool) (group string, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := RefreshUserGroupCache(id); err != nil {
					common.SysLog(common.LogText("failed to update user group cache: %s", err.Error()))
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		group, err := getUserGroupCache(id)
		if err == nil {
			return group, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select(commonGroupCol).Find(&group).Error
	if err != nil {
		return "", err
	}

	return group, nil
}

// GetUserSetting gets setting from Redis first, falls back to DB if needed
func GetUserSetting(id int, fromDB bool) (settingMap dto.UserSetting, err error) {
	var setting string
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserSettingCache(id, setting); err != nil {
					common.SysLog(common.LogText("failed to update user setting cache: %s", err.Error()))
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		setting, err := getUserSettingCache(id)
		if err == nil {
			return setting, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	// can be nil setting
	var safeSetting sql.NullString
	err = DB.Model(&User{}).Where("id = ?", id).Select("setting").Find(&safeSetting).Error
	if err != nil {
		return settingMap, err
	}
	if safeSetting.Valid {
		setting = safeSetting.String
	} else {
		setting = ""
	}
	userBase := &UserBase{
		Setting: setting,
	}
	return userBase.GetSetting(), nil
}

func IncreaseUserQuota(id int, quota int, db bool) (err error) {
	if quota < 0 {
		return errors.New("quota cannot be negative")
	}
	if err := common.ValidateWalletQuota(quota); err != nil {
		return err
	}
	if !db && common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUserQuota, id, quota)
		gopool.Go(func() {
			if err := cacheIncrUserQuota(id, int64(quota)); err != nil {
				common.SysLog(common.LogText("failed to increase user quota: %s", err.Error()))
			}
		})
		return nil
	}
	if err := increaseUserQuota(id, quota); err != nil {
		return err
	}
	gopool.Go(func() {
		if err := cacheIncrUserQuota(id, int64(quota)); err != nil {
			common.SysLog(common.LogText("failed to increase user quota: %s", err.Error()))
		}
	})
	return nil
}

func increaseUserQuota(id int, quota int) (err error) {
	result := DB.Model(&User{}).
		Where("id = ? AND quota <= ?", id, common.MaxWalletQuota-quota).
		Update("quota", gorm.Expr("quota + ?", quota))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var count int64
	if err := DB.Model(&User{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return ErrWalletQuotaLimitExceeded
}

func DecreaseUserQuota(id int, quota int, db bool) (err error) {
	if quota < 0 {
		return errors.New("quota cannot be negative")
	}
	gopool.Go(func() {
		err := cacheDecrUserQuota(id, int64(quota))
		if err != nil {
			common.SysLog(common.LogText("failed to decrease user quota: %s", err.Error()))
		}
	})
	if !db && common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUserQuota, id, -quota)
		return nil
	}
	return decreaseUserQuota(id, quota)
}

func decreaseUserQuota(id int, quota int) (err error) {
	err = DB.Model(&User{}).Where("id = ?", id).Update("quota", gorm.Expr("quota - ?", quota)).Error
	if err != nil {
		return err
	}
	return err
}

func DeltaUpdateUserQuota(id int, delta int) (err error) {
	if delta == 0 {
		return nil
	}
	if delta > 0 {
		return IncreaseUserQuota(id, delta, false)
	} else {
		return DecreaseUserQuota(id, -delta, false)
	}
}

//func GetRootUserEmail() (email string) {
//	DB.Model(&User{}).Where("role = ?", common.RoleRootUser).Select("email").Find(&email)
//	return email
//}

func GetRootUser() (user *User) {
	DB.Where("role = ?", common.RoleRootUser).First(&user)
	return user
}

func UpdateUserLastLogin(id int, ip string) {
	if normalized, err := NormalizeIPAddress(ip); err == nil {
		ip = normalized
	}
	if err := DB.Model(&User{}).Where("id = ?", id).Updates(map[string]interface{}{
		"last_login_at": common.GetTimestamp(),
		"last_login_ip": ip,
	}).Error; err != nil {
		common.SysLog("failed to update user last login: " + err.Error())
	}
}

func UpdateUserUsedQuotaAndRequestCount(id int, quota int) {
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUsedQuota, id, quota)
		addNewRecord(BatchUpdateTypeRequestCount, id, 1)
		return
	}
	updateUserUsedQuotaAndRequestCount(id, quota, 1)
}

// UpdateUserUsedQuota adjusts accumulated usage without changing request count.
func UpdateUserUsedQuota(id int, quota int) {
	if common.BatchUpdateEnabled {
		addNewRecord(BatchUpdateTypeUsedQuota, id, quota)
		return
	}
	if err := DB.Model(&User{}).Where("id = ?", id).Update("used_quota", gorm.Expr("used_quota + ?", quota)).Error; err != nil {
		common.SysLog(common.LogText("failed to update user used quota: %s", err.Error()))
	}
}

func updateUserUsedQuotaAndRequestCount(id int, quota int, count int) {
	err := DB.Model(&User{}).Where("id = ?", id).Updates(
		map[string]any{
			"used_quota":    gorm.Expr("used_quota + ?", quota),
			"request_count": gorm.Expr("request_count + ?", count),
		},
	).Error
	if err != nil {
		common.SysLog(common.LogText("failed to update user used quota and request count: %s", err.Error()))
		return
	}

	//// 更新缓存
	//if err := invalidateUserCache(id); err != nil {
	//	common.SysError("failed to invalidate user cache: " + err.Error())
	//}
}

func updateUserQuotaUsedQuotaAndRequestCount(id int, quota int, usedQuota int, requestCount int) {
	if quota == 0 && usedQuota == 0 && requestCount == 0 {
		return
	}

	err := DB.Model(&User{}).Where("id = ?", id).Updates(
		map[string]any{
			"quota":         gorm.Expr("quota + ?", quota),
			"used_quota":    gorm.Expr("used_quota + ?", usedQuota),
			"request_count": gorm.Expr("request_count + ?", requestCount),
		},
	).Error
	if err != nil {
		common.SysLog(common.LogText("failed to batch update user quota, used quota and request count: %s", err.Error()))
	}
}

// GetUsernameById gets username from Redis first, falls back to DB if needed
func GetUsernameById(id int, fromDB bool) (username string, err error) {
	defer func() {
		// Update Redis cache asynchronously on successful DB read
		if shouldUpdateRedis(fromDB, err) {
			gopool.Go(func() {
				if err := updateUserNameCache(id, username); err != nil {
					common.SysLog(common.LogText("failed to update user name cache: %s", err.Error()))
				}
			})
		}
	}()
	if !fromDB && common.RedisEnabled {
		username, err := getUserNameCache(id)
		if err == nil {
			return username, nil
		}
		// Don't return error - fall through to DB
	}
	fromDB = true
	err = DB.Model(&User{}).Where("id = ?", id).Select("username").Find(&username).Error
	if err != nil {
		return "", err
	}

	return username, nil
}

func IsLinuxDOIdAlreadyTaken(linuxDOId string) bool {
	var user User
	err := DB.Unscoped().Where("linux_do_id = ?", linuxDOId).First(&user).Error
	return !errors.Is(err, gorm.ErrRecordNotFound)
}

func (user *User) FillUserByLinuxDOId() error {
	if user.LinuxDOId == "" {
		return errors.New("linux do id is empty")
	}
	err := DB.Where("linux_do_id = ?", user.LinuxDOId).First(user).Error
	return err
}

func RootUserExists() bool {
	var user User
	err := DB.Where("role = ?", common.RoleRootUser).First(&user).Error
	if err != nil {
		return false
	}
	return true
}
