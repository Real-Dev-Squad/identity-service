package utils

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
)

// Logging Functions

func logProfileStored(client *firestore.Client, ctx context.Context, diff Diff, userId string, sessionId string) {
	newLog := Log{
		Type:      Constants["PROFILE_DIFF_STORED"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId":    userId,
			"sessionId": sessionId,
		},
		Body: map[string]interface{}{
			"userId":  userId,
			"profile": diffToMap(diff),
		},
	}

	client.Collection("logs").Add(ctx, newLog)
}

func LogProfileSkipped(client *firestore.Client, ctx context.Context, reason string, userId string, sessionId string) {
	newLog := Log{
		Type:      Constants["PROFILE_SKIPPED"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId":    userId,
			"sessionId": sessionId,
		},
		Body: map[string]interface{}{
			"userId": userId,
			"reason": reason,
		},
	}

	client.Collection("logs").Add(ctx, newLog)
}

func LogHealth(client *firestore.Client, ctx context.Context, userId string, isServiceRunning bool, sessionId string) {
	newLog := Log{
		Type:      Constants["PROFILE_SERVICE_HEALTH"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId":    userId,
			"sessionId": sessionId,
		},
		Body: map[string]interface{}{
			"userId":    	  userId,
			"serviceRunning": isServiceRunning,
		},
	}

	client.Collection("logs").Add(ctx, newLog)
}

func LogVerification(client *firestore.Client, ctx context.Context, status string, profileURL string, userId string) {
	var logtype string
	var logbody map[string]interface{}
	if status == "VERIFIED" {
		logtype = "PROFILE_VERIFIED"
		logbody = map[string]interface{}{
			"userId":     userId,
			"profileURL": profileURL,
		}
	} else if status == "BLOCKED" {
		logtype = "PROFILE_BLOCKED"
		logbody = map[string]interface{}{
			"userId": userId,
			"reason": "Chaincode not linked. Hash sent by service is not verified.",
		}
	}
	newLog := Log{
		Type:      logtype,
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: logbody,
	}
	client.Collection("logs").Add(ctx, newLog)
}
