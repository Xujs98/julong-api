package model

import (
	"slices"

	"gorm.io/gorm"
)

// UserLogScope is resolved by authorization before querying lists or details.
// Its predicates must be applied before pagination and counting, including when
// logs live in a separate database from users.
type UserLogScope struct {
	UserID          int
	ExcludedUserIDs []int
}

func (scope UserLogScope) Apply(query *gorm.DB) *gorm.DB {
	if scope.UserID > 0 {
		query = query.Where("user_id = ?", scope.UserID)
	}
	if len(scope.ExcludedUserIDs) > 0 {
		query = query.Where("user_id NOT IN ?", scope.ExcludedUserIDs)
	}
	return query
}

func (scope UserLogScope) Allows(userID int) bool {
	return (scope.UserID == 0 || scope.UserID == userID) && !slices.Contains(scope.ExcludedUserIDs, userID)
}
