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
			"serviceRunning": isServiceRunning,
		},
	}

	client.Collection("logs").Add(ctx, newLog)
}
