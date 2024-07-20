package utils

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Structures

type Log struct {
	Type      string                 `firestore:"type,omitempty"`
	Timestamp time.Time              `firestore:"timestamp,omitempty"`
	Meta      map[string]interface{} `firestore:"meta,omitempty"`
	Body      map[string]interface{} `firestore:"body,omitempty"`
}

type Res struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	YOE         int    `json:"yoe"`
	Company     string `json:"company"`
	Designation string `json:"designation"`
	GithubId    string `json:"github_id"`
	LinkedIn    string `json:"linkedin_id"`
	TwitterId   string `json:"twitter_id"`
	InstagramId string `json:"instagram_id"`
	Website     string `json:"website"`
}

type Diff struct {
	UserId      string    `firestore:"userId,omitempty"`
	Timestamp   time.Time `firestore:"timestamp,omitempty"`
	Approval    string    `firestore:"approval"`
	FirstName   string    `firestore:"first_name,omitempty"`
	LastName    string    `firestore:"last_name,omitempty"`
	Email       string    `firestore:"email,omitempty"`
	Phone       string    `firestore:"phone,omitempty"`
	YOE         int       `firestore:"yoe,omitempty"`
	Company     string    `firestore:"company,omitempty"`
	Designation string    `firestore:"designation,omitempty"`
	GithubId    string    `firestore:"github_id,omitempty"`
	LinkedIn    string    `firestore:"linkedin_id,omitempty"`
	TwitterId   string    `firestore:"twitter_id,omitempty"`
	InstagramId string    `firestore:"instagram_id,omitempty"`
	Website     string    `firestore:"website,omitempty"`
}

type Claims struct {
	jwt.RegisteredClaims
}

var Constants = map[string]string{
	"ENV_DEVELOPMENT":              "DEVELOPMENT",
	"ENV_PRODUCTION":               "PRODUCTION",
	"STORED":                       "stored",
	"FIRE_STORE_CRED":              "firestoreCred",
	"DISCORD_BOT_URL":              "discordBotURL",
	"IDENTITY_SERVICE_PRIVATE_KEY": "identityServicePrivateKey",
	"PROFILE_SERVICE_HEALTH":       "PROFILE_SERVICE_HEALTH",
	"PROFILE_SKIPPED":              "PROFILE_SKIPPED",
	"PROFILE_DIFF_STORED":          "PROFILE_DIFF_STORED",
	"STATUS_BLOCKED":               "BLOCKED",
	"PROFILE_SERVICE_BLOCKED":      "PROFILE_SERVICE_BLOCKED",
	"NOT_APPROVED":                 "NOT APPROVED",
	"PROFILE_SKIPPED_DUE_TO_UNAUTHENTICATED_ACCESS_TO_PROFILE_DATA": "profileSkippedDueToUnAuthenticatedAccessToProfileData",
	"PROFILE_SKIPPED_DUE_TO_ERROR_IN_GETTING_PROFILE_DATA":          "profileSkippedDueToErrorInGettingProfileData",
	"SKIPPED_SAME_LAST_REJECTED_DIFF":                               "skippedSameLastRejectedDiff",
	"SKIPPED_SAME_LAST_PENDING_DIFF":                                "skippedSameLastPendingDiff",
	"SKIPPED_CURRENT_USER_DATA_SAME_AS_DIFF":                        "skippedCurrentUserDataSameAsDiff",
	"SKIPPED_OTHER_ERROR":                                           "skippedOtherError",
	"SKIPPED_VALIDATION_ERROR":                                      "validation error",
}

func diffToMap(diff Diff) map[string]interface{} {
	return map[string]interface{}{
		"userId":       diff.UserId,
		"timestamp":    diff.Timestamp,
		"approval":     diff.Approval,
		"first_name":   diff.FirstName,
		"last_name":    diff.LastName,
		"email":        diff.Email,
		"phone":        diff.Phone,
		"yoe":          diff.YOE,
		"company":      diff.Company,
		"designation":  diff.Designation,
		"github_id":    diff.GithubId,
		"linkedin_id":  diff.LinkedIn,
		"twitter_id":   diff.TwitterId,
		"instagram_id": diff.InstagramId,
		"website":      diff.Website,
	}
}
