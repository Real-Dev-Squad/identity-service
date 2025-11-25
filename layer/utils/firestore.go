package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"golang.org/x/crypto/bcrypt"

	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Firestore Functions

func InitializeFirestoreClient(ctx context.Context) (*firestore.Client, error) {
	sa := option.WithCredentialsJSON([]byte(getParameter(Constants["FIRE_STORE_CRED"])))
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func getLastDiff(client *firestore.Client, ctx context.Context, userId string, approval string) (Res, string, error) {
	query := client.Collection("profileDiffs").Where("userId", "==", userId).Where("approval", "==", approval).OrderBy("timestamp", firestore.Desc).Limit(1).Documents(ctx)
	var lastdiff Diff
	var lastdiffId string
	for {
		Doc, err := query.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return Res{}, "", fmt.Errorf("failed to iterate profile diffs: %w", err)
		}
		err = Doc.DataTo(&lastdiff)
		if err != nil {
			return Res{}, "", fmt.Errorf("failed to convert diff data: %w", err)
		}
		lastdiffId = Doc.Ref.ID
	}
	return DiffToRes(lastdiff), lastdiffId, nil
}

func generateAndStoreDiff(client *firestore.Client, ctx context.Context, res Res, userId string, sessionId string) error {
	newDiff := resToDiff(res, userId)
	_, _, err := client.Collection("profileDiffs").Add(ctx, diffToMap(newDiff))
	if err != nil {
		return err
	}
	logProfileStored(client, ctx, newDiff, userId, sessionId)
	return nil
}

func SetNotApproved(client *firestore.Client, ctx context.Context, lastdiffId string) {
	client.Collection("profileDiffs").Doc(lastdiffId).Set(ctx, map[string]interface{}{
		"approval": Constants["NOT_APPROVED"],
	}, firestore.MergeAll)
}

func SetProfileStatusBlocked(client *firestore.Client, ctx context.Context, userId string, reason string, sessionId string, discordId string) {
	client.Collection("users").Doc(userId).Set(ctx, map[string]interface{}{
		"profileStatus": Constants["STATUS_BLOCKED"],
		"chaincode":     "",
		"updated_at":    time.Now().UnixMilli(),
	}, firestore.MergeAll)

	if discordId != "" {
		tokenString := generateJWTToken()
		postBody, _ := json.Marshal(map[string]string{
			"userId": discordId,
			"reason": reason,
		})

		responseBody := bytes.NewBuffer(postBody)

		discordURL := os.Getenv(Constants["DISCORD_BOT_URL"]) + "/profile/blocked"
		req, err := http.NewRequestWithContext(ctx, "POST", discordURL, responseBody)
		if err != nil {
			LogWarnWithError("Failed to create Discord bot request", err, map[string]interface{}{
				"function": "SetProfileStatusBlocked",
				"userId":   userId,
			})
		} else {
			req.Header.Add("Content-Type", "application/json")
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenString))
			
			// Create context with timeout and use client without timeout
			reqCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			req = req.WithContext(reqCtx)
			httpClient := &http.Client{} // No timeout - rely on context
			resp, err := httpClient.Do(req)
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
			if err != nil {
				LogWarnWithError("Failed to notify Discord bot", err, map[string]interface{}{
					"function": "SetProfileStatusBlocked",
					"userId":   userId,
				})
			}
		}
	}

	newLog := Log{
		Type:      Constants["PROFILE_SERVICE_BLOCKED"],
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

// Getdata retrieves and processes profile data from user service
// Returns an error if there's a problem, or nil if successful
// If the profile should be skipped (same data, etc.), it returns nil but logs the skip reason
func Getdata(client *firestore.Client, ctx context.Context, userId string, userUrl string, chaincode string, userData Res, sessionId string, discordId string) error {
	userUrl = userUrl + "profile"
	hashedChaincode, err := bcrypt.GenerateFromPassword([]byte(chaincode), bcrypt.DefaultCost)
	if err != nil {
		errMsg := fmt.Sprintf("chaincode encryption failed: %v", err)
		LogProfileSkipped(client, ctx, errMsg, userId, sessionId)
		SetProfileStatusBlocked(client, ctx, userId, errMsg, sessionId, discordId)
		return fmt.Errorf("chaincode not encrypted: %w", err)
	}

	// Create context with timeout and use client without timeout
	reqCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, "GET", userUrl, nil)
	if err != nil {
		errMsg := fmt.Sprintf("failed to create request: %v", err)
		LogProfileSkipped(client, ctx, errMsg, userId, sessionId)
		SetProfileStatusBlocked(client, ctx, userId, errMsg, sessionId, discordId)
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", string(hashedChaincode)))
	httpClient := &http.Client{} // No timeout - rely on context
	resp, err := httpClient.Do(req)
	if err != nil {
		errMsg := fmt.Sprintf("failed to get profile data: %v", err)
		LogProfileSkipped(client, ctx, errMsg, userId, sessionId)
		SetProfileStatusBlocked(client, ctx, userId, errMsg, sessionId, discordId)
		return fmt.Errorf("error getting profile data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		errMsg := "Unauthenticated Access to Profile Data"
		LogProfileSkipped(client, ctx, errMsg, userId, sessionId)
		SetProfileStatusBlocked(client, ctx, userId, errMsg, sessionId, discordId)
		return NewProfileError("UNAUTHENTICATED", errMsg, 401, nil)
	}
	if resp.StatusCode != 200 {
		errMsg := "Error in getting Profile Data"
		LogProfileSkipped(client, ctx, errMsg, userId, sessionId)
		SetProfileStatusBlocked(client, ctx, userId, errMsg, sessionId, discordId)
		return fmt.Errorf("error in getting profile data: status code %d", resp.StatusCode)
	}

	r, err := io.ReadAll(resp.Body)
	if err != nil {
		errMsg := fmt.Sprintf("failed to read response: %v", err)
		LogProfileSkipped(client, ctx, errMsg, userId, sessionId)
		SetProfileStatusBlocked(client, ctx, userId, errMsg, sessionId, discordId)
		return fmt.Errorf("error reading profile data: %w", err)
	}
	var res Res
	err = json.Unmarshal([]byte(r), &res)
	if err != nil {
		errMsg := fmt.Sprintf("failed to unmarshal JSON: %v", err)
		LogProfileSkipped(client, ctx, errMsg, userId, sessionId)
		SetProfileStatusBlocked(client, ctx, userId, errMsg, sessionId, discordId)
		return fmt.Errorf("error converting data to json: %w", err)
	}

	err = res.Validate()
	if err != nil {
		errMsg := fmt.Sprintf("validation failed: %v", err)
		LogProfileSkipped(client, ctx, errMsg, userId, sessionId)
		SetProfileStatusBlocked(client, ctx, userId, errMsg, sessionId, discordId)
		return fmt.Errorf("error in validation: %w", err)
	}

	lastPendingDiff, lastPendingDiffId, err := getLastDiff(client, ctx, userId, "PENDING")
	if err != nil {
		// Log error but continue processing
		LogWarnWithError("Failed to get last pending diff", err, map[string]interface{}{
			"function": "Getdata",
			"userId":   userId,
		})
	}

	if lastPendingDiff != res && userData != res {
		if lastPendingDiffId != "" {
			SetNotApproved(client, ctx, lastPendingDiffId)
		}
		lastRejectedDiff, lastRejectedDiffId, err := getLastDiff(client, ctx, userId, Constants["NOT_APPROVED"])
		if err != nil {
			LogWarnWithError("Failed to get last rejected diff", err, map[string]interface{}{
				"function": "Getdata",
				"userId":   userId,
			})
		}
		if lastRejectedDiff != res {
			err = generateAndStoreDiff(client, ctx, res, userId, sessionId)
			if err != nil {
				return fmt.Errorf("failed to generate and store diff: %w", err)
			}
		} else {
			LogProfileSkipped(client, ctx, "Last Rejected Diff is same as New Profile Data. Rejected Diff Id: "+lastRejectedDiffId, userId, sessionId)
			// This is not an error, just a skip reason
			return nil
		}
	} else if userData == res {
		LogProfileSkipped(client, ctx, "Current User Data is same as New Profile Data", userId, sessionId)
		if lastPendingDiffId != "" {
			SetNotApproved(client, ctx, lastPendingDiffId)
		}
		// This is not an error, just a skip reason
		return nil
	} else {
		LogProfileSkipped(client, ctx, "Last Pending Diff is same as New Profile Data", userId, sessionId)
		// This is not an error, just a skip reason
		return nil
	}

	return nil
}

func GetDataFromBody(body []byte) (string, string) {
	type extractedBody struct {
		UserId    string `json:"userId"`
		SessionId string `json:"sessionId"`
	}

	var e extractedBody
	json.Unmarshal(body, &e)
	return e.UserId, e.SessionId
}

func GenerateHealthMessage() string {
	return "Awesome, Server health is good!!!"
}

/*
Function to extract userId from the request body
*/

func GetUserIdFromBody(body []byte) string {
	type extractedBody struct {
		UserId string `json:"userId"`
	}

	var e extractedBody
	json.Unmarshal(body, &e)
	return e.UserId
}

/*
Function to get the userData using userId
*/

func GetUserData(client *firestore.Client, ctx context.Context, userId string) (string, string, string, error) {
	dsnap, err := client.Collection("users").Doc(userId).Get(ctx)
	if err != nil {
		return "", "", "", err
	}

	var user User
	err = dsnap.DataTo(&user)
	if err != nil {
		data := dsnap.Data()
		
		if profileURLVal, exists := data["profileURL"]; exists && profileURLVal != nil {
			if _, ok := profileURLVal.(string); !ok {
				return "", "", "", errors.New("profile url is not a string")
			}
		} else {
			return "", "", "", errors.New("profile url is not a string")
		}
		
		if chaincodeVal, exists := data["chaincode"]; exists && chaincodeVal != nil {
			if _, ok := chaincodeVal.(string); !ok {
				return "", "", "", errors.New("chaincode is not a string")
			}
		} else {
			return "", "", "", errors.New("chaincode is not a string")
		}
		
		return "", "", "", fmt.Errorf("failed to convert user data: %w", err)
	}

	if user.ProfileURL == "" {
		return "", "", "", errors.New("profile url is not a string")
	}

	if user.Chaincode == "" {
		newLog := Log{
			Type:      "VERIFICATION_BLOCKED",
			Timestamp: time.Now(),
			Meta: map[string]interface{}{
				"userId": userId,
			},
			Body: map[string]interface{}{
				"userId": userId,
				"reason": "Chaincode is empty. Generate new one.",
			},
		}
		logCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		_, _, _ = client.Collection("logs").Add(logCtx, newLog)
		cancel()
		return "", "", "", errors.New("chaincode is blocked")
	}

	return user.ProfileURL, user.ProfileStatus, user.Chaincode, nil
}

/*
Function for setting the profileStatus in user object in firestore
*/
func SetProfileStatus(client *firestore.Client, ctx context.Context, id string, status string) error {
	var newData = map[string]interface{}{
		"profileStatus": status,
	}

	if status == "BLOCKED" {
		newData = map[string]interface{}{
			"profileStatus": status,
			"chaincode":     "",
			"updated_at":    time.Now().UnixMilli(),
		}
	}

	_, err := client.Collection("users").Doc(id).Set(ctx, newData, firestore.MergeAll)

	if err != nil {
		return errors.New("unable to set profile status")
	}

	return nil
}
