package dbmodel

import (
	"database/sql"
	"time"

	dbsqlc "backend/internal/db/sqlc"
	"backend/internal/repository/model"
)

func NullString(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *value, Valid: true}
}

func StringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	return &value.String
}

func NullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func TimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}

func BoolInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func IntBool(value int64) bool {
	return value != 0
}

func UserFromByUsername(row dbsqlc.UserByUsernameRow) model.User {
	return model.User{Id: row.ID, Username: row.Username, PasswordHash: row.PasswordHash, Status: row.Status, OAuthProvider: row.OauthProvider, OAuthProviderId: row.OauthProviderID, Email: StringPtr(row.Email), AuthSource: row.AuthSource, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, LastLoginAt: TimePtr(row.LastLoginAt)}
}

func UserFromByID(row dbsqlc.UserByIDRow) model.User {
	return model.User{Id: row.ID, Username: row.Username, PasswordHash: row.PasswordHash, Status: row.Status, OAuthProvider: row.OauthProvider, OAuthProviderId: row.OauthProviderID, Email: StringPtr(row.Email), AuthSource: row.AuthSource, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, LastLoginAt: TimePtr(row.LastLoginAt)}
}

func UserFromByEmail(row dbsqlc.UserByEmailRow) model.User {
	return model.User{Id: row.ID, Username: row.Username, PasswordHash: row.PasswordHash, Status: row.Status, OAuthProvider: row.OauthProvider, OAuthProviderId: row.OauthProviderID, Email: StringPtr(row.Email), AuthSource: row.AuthSource, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, LastLoginAt: TimePtr(row.LastLoginAt)}
}

func UserFromProjectMember(row dbsqlc.ProjectMembersRow) model.User {
	return model.User{Id: row.ID, Username: row.Username, PasswordHash: row.PasswordHash, Status: row.Status, OAuthProvider: row.OauthProvider, OAuthProviderId: row.OauthProviderID, Email: StringPtr(row.Email), AuthSource: row.AuthSource, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, LastLoginAt: TimePtr(row.LastLoginAt)}
}

func RoleFromSQLC(row dbsqlc.Role) model.Role {
	return model.Role{Id: row.ID, Code: row.Code, Name: row.Name, Description: StringPtr(row.Description), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func PermissionFromSQLC(row dbsqlc.Permission) model.Permission {
	return model.Permission{Id: row.ID, Code: row.Code, Name: row.Name, Description: StringPtr(row.Description), CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func ProjectFromByID(row dbsqlc.ProjectByIDRow) model.Project {
	return model.Project{Id: row.ID, Name: row.Name, Code: row.Code, IsActive: row.IsActive, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func ProjectFromByCode(row dbsqlc.ProjectByCodeRow) model.Project {
	return model.Project{Id: row.ID, Name: row.Name, Code: row.Code, IsActive: row.IsActive, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func ProjectFromList(row dbsqlc.ListProjectsByMemberRow) model.Project {
	return model.Project{Id: row.ID, Name: row.Name, Code: row.Code, IsActive: row.IsActive, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func ProjectFromActiveList(row dbsqlc.ListActiveProjectsByMemberRow) model.Project {
	return model.Project{Id: row.ID, Name: row.Name, Code: row.Code, IsActive: row.IsActive, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}
