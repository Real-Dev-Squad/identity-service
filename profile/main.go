package main

import (
	"context"
	"encoding/json"
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

type profileStatusCount struct {
	totalProfilesChecked                          int
	profilesSkippedDueToProfileURL                int
	profilesSkippedDueToServiceDown               int
	profilesSkippedDueToCurrentUserDataSameAsDiff int
	profilesSkippedDueToSameLastRejectedDiff      int
	profilesSkippedSameLastPendingDiff            int
	profileDiffsStored                            int
}

// var profileDiffsStored, profilesSkippedDueToProfileURL, profilesSkippedDueToServiceDown, profilesSkippedDueToCurrentUserDataSameAsDiff, profilesSkippedDueToSameLastRejectedDiff, profilesSkippedSameLastPendingDiff []string

type profileStatusArray struct {
	profileDiffsStored                            []string
	profilesSkippedDueToProfileURL                []string
	profilesSkippedDueToServiceDown               []string
	profilesSkippedDueToCurrentUserDataSameAsDiff []string
	profilesSkippedDueToSameLastRejectedDiff      []string
	profilesSkippedSameLastPendingDiff            []string
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

/*
 Function for setting the profileStatus in user object in firestore
*/
func setProfileStatusBlocked(client *firestore.Client, ctx context.Context, userId string, status string) {
	client.Collection("users").Doc(userId).Set(ctx, map[string]interface{}{
		"profileStatus": status,
	}, firestore.MergeAll)

	newLog := Log{
		Type:      "PROFILE_BLOCKED",
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: map[string]interface{}{
			"userId": userId,
			"reason": "service not running",
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
	}
}

/*
 Getting data from the user's service
*/
func getdata(client *firestore.Client, ctx context.Context, userId string, userUrl string) string {
	var status string = "stored"
	userUrl = userUrl + "profile"
	resp, err := http.Get(userUrl)
	if err != nil {
		log.Fatal(err)
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
	return status
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

	profileStatusCounter := profileStatusCount{0, 0, 0, 0, 0, 0, 0}
	profileStatusList := profileStatusArray{}

	iter := client.Collection("users").Where("profileStatus", "==", "VERIFIED").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		profileStatusCounter.totalProfilesChecked += 1
		var userId string = doc.Ref.ID
		var userUrl string
		if str, ok := doc.Data()["profileURL"].(string); ok {
			userUrl = str
		} else {
			profileStatusCounter.profilesSkippedDueToProfileURL += 1
			profileStatusList.profilesSkippedDueToProfileURL = append(profileStatusList.profilesSkippedDueToProfileURL, userId)
			logProfileSkipped(client, ctx, userId, "Profile URL not available")
			continue
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
			profileStatusCounter.profilesSkippedDueToServiceDown += 1
			profileStatusList.profilesSkippedDueToServiceDown = append(profileStatusList.profilesSkippedDueToServiceDown, userId)
			logProfileSkipped(client, ctx, userId, "Service Down")
			setProfileStatusBlocked(client, ctx, userId, "service not running")
			continue
		}

		status := getdata(client, ctx, userId, userUrl)
		if status == "skippedSameLastPendingDiff" {
			profileStatusCounter.profilesSkippedSameLastPendingDiff += 1
			profileStatusList.profilesSkippedSameLastPendingDiff = append(profileStatusList.profilesSkippedSameLastPendingDiff, userId)
		} else if status == "skippedCurrentUserDataSameAsDiff" {
			profileStatusCounter.profilesSkippedDueToCurrentUserDataSameAsDiff += 1
			profileStatusList.profilesSkippedDueToCurrentUserDataSameAsDiff = append(profileStatusList.profilesSkippedDueToCurrentUserDataSameAsDiff, userId)
		} else if status == "skippedSameLastRejectedDiff" {
			profileStatusCounter.profilesSkippedDueToSameLastRejectedDiff += 1
			profileStatusList.profilesSkippedDueToSameLastRejectedDiff = append(profileStatusList.profilesSkippedDueToSameLastRejectedDiff, userId)
		} else {
			profileStatusCounter.profileDiffsStored += 1
			profileStatusList.profileDiffsStored = append(profileStatusList.profileDiffsStored, userId)
		}
	}

	var report = map[string]interface{}{
		"TotalProfilesChecked": profileStatusCounter.totalProfilesChecked,
		"Stored": map[string]interface{}{
			"count":   profileStatusCounter.profileDiffsStored,
			"userIds": profileStatusList.profileDiffsStored,
		},
		"Skipped": map[string]interface{}{
			"CurrentUserDataSameAsDiff": map[string]interface{}{
				"count":   profileStatusCounter.profilesSkippedDueToCurrentUserDataSameAsDiff,
				"userIds": profileStatusList.profilesSkippedDueToCurrentUserDataSameAsDiff,
			},
			"SameLastRejectedDiff": map[string]interface{}{
				"count":   profileStatusCounter.profilesSkippedDueToSameLastRejectedDiff,
				"userIds": profileStatusList.profilesSkippedDueToSameLastRejectedDiff,
			},
			"NoProfileURLCount": map[string]interface{}{
				"count":   profileStatusCounter.profilesSkippedDueToProfileURL,
				"userIds": profileStatusList.profilesSkippedDueToProfileURL,
			},
			"ServiceDown": map[string]interface{}{
				"count":   profileStatusCounter.profilesSkippedDueToServiceDown,
				"userIds": profileStatusList.profilesSkippedDueToServiceDown,
			},
			"SameLastPendingDiff": map[string]interface{}{
				"count":   profileStatusCounter.profilesSkippedSameLastPendingDiff,
				"userIds": profileStatusList.profilesSkippedSameLastPendingDiff,
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
