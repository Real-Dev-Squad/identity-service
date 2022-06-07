package main

import (
	// "errors"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

/*
 Structures
*/
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

type ProfileReport struct {
	TotalProfilesChecked int                    `json:"totalProfilesChecked"`
	ProfileStored        map[string]interface{} `json:"profilesChecked"`
	ProfilesBlocked      map[string]interface{} `json:"profilesBlocked"`
}

/*
 Structures Conversions
*/
func diffToRes(diff Diff) Res {
	return Res{
		FirstName:   diff.FirstName,
		LastName:    diff.LastName,
		Email:       diff.Email,
		Phone:       diff.Phone,
		YOE:         diff.YOE,
		Company:     diff.Company,
		Designation: diff.Designation,
		GithubId:    diff.GithubId,
		LinkedIn:    diff.LinkedIn,
		TwitterId:   diff.TwitterId,
		InstagramId: diff.InstagramId,
		Website:     diff.Website,
	}
}

func resToDiff(res Res, userId string) Diff {
	return Diff{
		UserId:      userId,
		Timestamp:   time.Now(),
		Approval:    "PENDING",
		FirstName:   res.FirstName,
		LastName:    res.LastName,
		Email:       res.Email,
		Phone:       res.Phone,
		YOE:         res.YOE,
		Company:     res.Company,
		Designation: res.Designation,
		GithubId:    res.GithubId,
		LinkedIn:    res.LinkedIn,
		TwitterId:   res.TwitterId,
		InstagramId: res.InstagramId,
		Website:     res.Website,
	}
}

/*
 Setting Firestore Key for development/production
*/
func getFirestoreKey() string {
	if os.Getenv(("environment")) == "DEVELOPMENT" {
		return os.Getenv("firestoreCred")
	} else if os.Getenv(("environment")) == "PRODUCTION" {
		var parameterName string = "firestoreCred"

		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		svc := ssm.New(sess)

		results, err := svc.GetParameter(&ssm.GetParameterInput{
			Name: &parameterName,
		})
		if err != nil {
			log.Fatalf(err.Error())
		}

		return *results.Parameter.Value
	} else {
		return ""
	}
}

/*
 Utils
*/

/*
 Function to initialize the firestore client
*/
func initializeFirestoreClient(ctx context.Context) (*firestore.Client, error) {
	sa := option.WithCredentialsJSON([]byte(getFirestoreKey()))
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

/*
 MODELS
*/

/*
 Logs the health of the user's service
*/
func logHealth(client *firestore.Client, ctx context.Context, userId string, isServiceRunning bool) {
	newLog := Log{
		Type:      "SERVICE_HEALTH",
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: map[string]interface{}{
			"userId":         userId,
			"serviceRunning": isServiceRunning,
		},
	}
	client.Collection("logs").Add(ctx, newLog)
}

/*
 Logs the status of the user's profileDiff
*/
func logProfileSkipped(client *firestore.Client, ctx context.Context, userId string, reason string) {
	newLog := Log{
		Type:      "PROFILE_SKIPPED",
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: map[string]interface{}{
			"userId": userId,
			"reason": reason,
		},
	}
	client.Collection("logs").Add(ctx, newLog)
}

func logProfileStored(client *firestore.Client, ctx context.Context, userId string) {
	newLog := Log{
		Type:      "PROFILE_DIFF_STORED",
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: map[string]interface{}{
			"userId": userId,
		},
	}
	client.Collection("logs").Add(ctx, newLog)
}

/*
 Function for setting the profileStatus in user object in firestore
*/
func setProfileStatusBlocked(client *firestore.Client, ctx context.Context, userId string, reason string) {
	client.Collection("users").Doc(userId).Set(ctx, map[string]interface{}{
		"profileStatus": "BLOCKED",
		"chaincode":     "",
	}, firestore.MergeAll)

	newLog := Log{
		Type:      "PROFILE_BLOCKED",
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: map[string]interface{}{
			"userId": userId,
			"reason": reason,
		},
	}
	client.Collection("logs").Add(ctx, newLog)
}

/*
 sets the user's profile diff to not approved
*/
func setNotApproved(client *firestore.Client, ctx context.Context, lastdiffId string) {
	client.Collection("profileDiffs").Doc(lastdiffId).Set(ctx, map[string]interface{}{
		"approval": "NOT APPROVED",
	}, firestore.MergeAll)
}

/*
 Get the last profile diff of the user
*/
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
	return diffToRes(lastdiff), lastdiffId
}

/*
 Get the user's profile data
*/
func getUserData(client *firestore.Client, ctx context.Context, userId string) Res {
	dsnap, err := client.Collection("users").Doc(userId).Get(ctx)
	if err != nil {
		log.Fatal(err)
	}
	var userData Diff
	err = dsnap.DataTo(&userData)
	if err != nil {
		log.Fatal(err)
	}

	return diffToRes(userData)
}

/*
 Generate and Store Profile Diff
*/
func generateAndStoreDiff(client *firestore.Client, ctx context.Context, res Res, userId string) {
	var diff Diff = resToDiff(res, userId)
	_, _, err := client.Collection("profileDiffs").Add(ctx, diff)
	if err != nil {
		log.Fatal(err)
	} else {
		logProfileStored(client, ctx, userId)
	}
}

/*
 Getting data from the user's service
*/
func getdata(client *firestore.Client, ctx context.Context, userId string, userUrl string, chaincode string) (string, error) {
	var status string = "stored"
	userUrl = userUrl + "profile"

	hashedChaincode, err := bcrypt.GenerateFromPassword([]byte(chaincode), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	postBody, _ := json.Marshal(map[string]string{
		"hash": string(hashedChaincode),
	})

	reqBody := bytes.NewBuffer(postBody)

	resp, err := http.Post(userUrl, "application/json", reqBody)
	if err != nil {
		log.Fatal(err)
	}

	if resp.StatusCode == 401 {
		return "", errors.New("unauthenticated access")
	}

	defer resp.Body.Close()

	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	var res Res
	json.Unmarshal([]byte(r), &res)

	lastPendingDiff, lastPendingDiffId := getLastDiff(client, ctx, userId, "PENDING")
	userData := getUserData(client, ctx, userId)
	if lastPendingDiff != res && userData != res {
		if lastPendingDiffId != "" {
			setNotApproved(client, ctx, lastPendingDiffId)
		}
		lastRejectedDiff, lastRejectedDiffId := getLastDiff(client, ctx, userId, "NOT APPROVED")
		if lastRejectedDiff != res {
			generateAndStoreDiff(client, ctx, res, userId)
		} else {
			status = "skippedSameLastRejectedDiff"
			logProfileSkipped(client, ctx, userId, "Last Rejected Diff is same as New Profile Data. Rejected Diff Id: "+lastRejectedDiffId)
		}
	} else if userData == res {
		status = "skippedCurrentUserDataSameAsDiff"
		logProfileSkipped(client, ctx, userId, "Current User Data is same as New Profile Data")
		if lastPendingDiffId != "" {
			setNotApproved(client, ctx, lastPendingDiffId)
		}
	} else {
		status = "skippedSameLastPendingDiff"
		logProfileSkipped(client, ctx, userId, "Last Pending Diff is same as New Profile Data")
	}
	return status, nil
}

/*
 Controller
*/
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()
	client, err := initializeFirestoreClient(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	var totalProfilesChecked, profilesSkippedDueToProfileURLCount, profilesSkippedDueToServiceDownCount, profilesSkippedDueToCurrentUserDataSameAsDiffCount, profilesSkippedDueToSameLastRejectedDiffCount, profilesSkippedSameLastPendingDiffCount, profileDiffsStoredCount int = 0, 0, 0, 0, 0, 0, 0
	var profileDiffsStored, profilesSkippedDueToProfileURL, profilesSkippedDueToServiceDown, profilesSkippedDueToCurrentUserDataSameAsDiff, profilesSkippedDueToSameLastRejectedDiff, profilesSkippedSameLastPendingDiff []string

	iter := client.Collection("users").Where("profileStatus", "==", "VERIFIED").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		totalProfilesChecked = totalProfilesChecked + 1
		var userId string = doc.Ref.ID
		var userUrl string
		var chaincode string
		if str, ok := doc.Data()["profileURL"].(string); ok {
			userUrl = str
		} else {
			profilesSkippedDueToProfileURLCount = profilesSkippedDueToProfileURLCount + 1
			profilesSkippedDueToProfileURL = append(profilesSkippedDueToProfileURL, userId)
			logProfileSkipped(client, ctx, userId, "Profile URL not available")
			continue
		}

		if str, ok := doc.Data()["chaincode"].(string); ok {
			chaincode = str
		}

		if userUrl[len(userUrl)-1] != '/' {
			userUrl = userUrl + "/"
		}
		var isServiceRunning bool
		_, err = http.Get(userUrl + "health")
		if err != nil {
			isServiceRunning = false
		} else {
			isServiceRunning = true
		}

		logHealth(client, ctx, userId, isServiceRunning)
		if !isServiceRunning {
			profilesSkippedDueToServiceDownCount = profilesSkippedDueToServiceDownCount + 1
			profilesSkippedDueToServiceDown = append(profilesSkippedDueToServiceDown, userId)
			logProfileSkipped(client, ctx, userId, "Service Down")
			setProfileStatusBlocked(client, ctx, userId, "service not running")
			continue
		}

		status, err := getdata(client, ctx, userId, userUrl, chaincode)
		if err != nil {
			setProfileStatusBlocked(client, ctx, userId, "Unauthenticated Profile Access")
			logProfileSkipped(client, ctx, userId, "Unauthenticated Profile Access")
			continue
		}

		if status == "skippedSameLastPendingDiff" {
			profilesSkippedSameLastPendingDiffCount = profilesSkippedSameLastPendingDiffCount + 1
			profilesSkippedSameLastPendingDiff = append(profilesSkippedSameLastPendingDiff, userId)
		} else if status == "skippedCurrentUserDataSameAsDiff" {
			profilesSkippedDueToCurrentUserDataSameAsDiffCount = profilesSkippedDueToCurrentUserDataSameAsDiffCount + 1
			profilesSkippedDueToCurrentUserDataSameAsDiff = append(profilesSkippedDueToCurrentUserDataSameAsDiff, userId)
		} else if status == "skippedSameLastRejectedDiff" {
			profilesSkippedDueToSameLastRejectedDiffCount = profilesSkippedDueToSameLastRejectedDiffCount + 1
			profilesSkippedDueToSameLastRejectedDiff = append(profilesSkippedDueToSameLastRejectedDiff, userId)
		} else {
			profileDiffsStoredCount = profileDiffsStoredCount + 1
			profileDiffsStored = append(profileDiffsStored, userId)
		}

	}

	var report = map[string]interface{}{
		"TotalProfilesChecked": totalProfilesChecked,
		"Stored": map[string]interface{}{
			"count":   profileDiffsStoredCount,
			"userIds": profileDiffsStored,
		},
		"Skipped": map[string]interface{}{
			"CurrentUserDataSameAsDiff": map[string]interface{}{
				"count":   profilesSkippedDueToCurrentUserDataSameAsDiffCount,
				"userIds": profilesSkippedDueToCurrentUserDataSameAsDiff,
			},
			"SameLastRejectedDiff": map[string]interface{}{
				"count":   profilesSkippedDueToSameLastRejectedDiffCount,
				"userIds": profilesSkippedDueToSameLastRejectedDiff,
			},
			"NoProfileURLCount": map[string]interface{}{
				"count":   profilesSkippedDueToProfileURLCount,
				"userIds": profilesSkippedDueToProfileURL,
			},
			"ServiceDown": map[string]interface{}{
				"count":   profilesSkippedDueToServiceDownCount,
				"userIds": profilesSkippedDueToServiceDown,
			},
			"SameLastPendingDiff": map[string]interface{}{
				"count":   profilesSkippedSameLastPendingDiffCount,
				"userIds": profilesSkippedSameLastPendingDiff,
			},
		},
	}
	reportjson, err := json.Marshal(report)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	defer client.Close()

	return events.APIGatewayProxyResponse{
		Body:       string(reportjson),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
