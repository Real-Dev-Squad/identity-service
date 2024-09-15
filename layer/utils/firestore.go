package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
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

func getLastDiff(client *firestore.Client, ctx context.Context, userId string, approval string) (Res, string) {
	query := client.Collection("profileDiffs").Where("userId", "==", userId).Where("approval", "==", approval).OrderBy("timestamp", firestore.Desc).Limit(1).Documents(ctx)
	var lastdiff Diff
	var lastdiffId string
	for {
		Doc, err := query.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		err = Doc.DataTo(&lastdiff)
		if err != nil {
			log.Fatal(err)
		}
		lastdiffId = Doc.Ref.ID
	}
	return DiffToRes(lastdiff), lastdiffId
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

		httpClient := &http.Client{}
		req, _ := http.NewRequest("POST", os.Getenv(Constants["DISCORD_BOT_URL"])+"/profile/blocked", responseBody)
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenString))
		httpClient.Do(req)
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

func Getdata(client *firestore.Client, ctx context.Context, userId string, userUrl string, chaincode string, userData Res, sessionId string, discordId string) string {
	var status string = ""
	userUrl = userUrl + "profile"
	hashedChaincode, err := bcrypt.GenerateFromPassword([]byte(chaincode), bcrypt.DefaultCost)
	if err != nil {
		LogProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		SetProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return "chaincode not encrypted"
	}

	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", userUrl, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", string(hashedChaincode)))
	resp, err := httpClient.Do(req)
	if err != nil {
		LogProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		SetProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return "error getting profile data"
	}
	if resp.StatusCode == 401 {
		LogProfileSkipped(client, ctx, userId, "Unauthenticated Access to Profile Data", sessionId)
		SetProfileStatusBlocked(client, ctx, userId, "Unauthenticated Access to Profile Data", sessionId, discordId)
		resp.Body.Close()
		return "unauthenticated access to profile data"
	}
	if resp.StatusCode != 200 {
		LogProfileSkipped(client, ctx, userId, "Error in getting Profile Data", sessionId)
		SetProfileStatusBlocked(client, ctx, userId, "Error in getting Profile Data", sessionId, discordId)
		resp.Body.Close()
		return "error in getting profile data"
	}

	defer resp.Body.Close()

	r, err := io.ReadAll(resp.Body)
	if err != nil {
		LogProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		SetProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return "error reading profile data"
	}
	var res Res
	err = json.Unmarshal([]byte(r), &res)
	if err != nil {
		LogProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		SetProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return "error converting data to json"
	}

	err = res.Validate()

	if err != nil {
		LogProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		SetProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return fmt.Sprintf("error in validation: ", err)
	}

	lastPendingDiff, lastPendingDiffId := getLastDiff(client, ctx, userId, "PENDING")
	if lastPendingDiff != res && userData != res {
		if lastPendingDiffId != "" {
			SetNotApproved(client, ctx, lastPendingDiffId)
		}
		lastRejectedDiff, lastRejectedDiffId := getLastDiff(client, ctx, userId, Constants["NOT_APPROVED"])
		if lastRejectedDiff != res {
			generateAndStoreDiff(client, ctx, res, userId, sessionId)
		} else {
			status = "same last rejected diff " + lastRejectedDiffId
			LogProfileSkipped(client, ctx, userId, "Last Rejected Diff is same as New Profile Data. Rejected Diff Id: "+lastRejectedDiffId, sessionId)
		}
	} else if userData == res {
		status = "same data exists"
		LogProfileSkipped(client, ctx, userId, "Current User Data is same as New Profile Data", sessionId)
		if lastPendingDiffId != "" {
			SetNotApproved(client, ctx, lastPendingDiffId)
		}
	} else {
		status = "same last pending diff"
		LogProfileSkipped(client, ctx, userId, "Last Pending Diff is same as New Profile Data", sessionId)
	}
	return status
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
	var profileURL string
	var profileStatus string
	var chaincode string
	if err != nil {
		return "", "", "", err
	}
	if str, ok := dsnap.Data()["profileURL"].(string); ok {
		profileURL = str
	} else {
		return "", "", "", errors.New("profile url is not a string")
	}
	if str, ok := dsnap.Data()["profileStatus"].(string); ok {
		profileStatus = str
	} else {
		profileStatus = ""
	}

	if str, ok := dsnap.Data()["chaincode"].(string); ok {
		if str != "" {
			chaincode = str
		} else {
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
			client.Collection("logs").Add(ctx, newLog)
			return "", "", "", errors.New("chaincode is blocked")
		}
	} else {
		return "", "", "", errors.New("chaincode is not a string")
	}

	return profileURL, profileStatus, chaincode, nil
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
