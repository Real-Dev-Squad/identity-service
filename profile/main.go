package main

import (
	// "errors"
	"context"
	"encoding/json"
	"fmt"
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
	Username    string    `firestore:"username,omitempty"`
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

type Chaincode struct {
	Username  string    `firestore:"username,omitempty"`
	Timestamp time.Time `firestore:"timestamp,omitempty"`
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

func resToDiff(res Res, username string) Diff {
	return Diff{
		Username:    username,
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
func logHealth(client *firestore.Client, ctx context.Context, username string, isServiceRunning bool) {
	var logdata = map[string]interface{}{
		"username":       username,
		"serviceRunning": isServiceRunning,
	}
	newLog := Log{
		Type:      "PROFILE_HEALTH",
		Timestamp: time.Now(),
		Meta:      logdata,
		Body:      logdata,
	}
	client.Collection("logs").Add(ctx, newLog)
}

/*
 Function for setting the identityStatus in user object in firestore
*/
func setIdentityStatus(client *firestore.Client, ctx context.Context, id string, status string, username string) {
	client.Collection("users").Doc(id).Set(ctx, map[string]interface{}{
		"identityStatus": status,
	}, firestore.MergeAll)

	newLog := Log{
		Type:      "PROFILE_BLOCKED",
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"username": username,
		},
		Body: map[string]interface{}{
			"username": username,
			"reason":   "service not running",
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
func getLastDiff(client *firestore.Client, ctx context.Context, username string) (Res, string) {
	query := client.Collection("profileDiffs").Where("username", "==", username).Where("approval", "==", "PENDING").OrderBy("timestamp", firestore.Desc).Limit(1).Documents(ctx)
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
func getUserData(client *firestore.Client, ctx context.Context, username string) Res {
	query := client.Collection("users").Where("username", "==", username).Limit(1).Documents(ctx)
	var userData Diff
	for {
		Doc, err := query.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		err = Doc.DataTo(&userData)
		if err != nil {
			log.Fatal(err)
		}
	}
	return diffToRes(userData)
}

/*
 Generate and Store Profile Diff
*/
func generateAndStoreDiff(client *firestore.Client, ctx context.Context, res Res, username string) {
	var diff Diff = resToDiff(res, username)
	_, _, err := client.Collection("profileDiffs").Add(ctx, diff)
	if err != nil {
		log.Fatal(err)
	}
}

/*
 Getting data from the user's service
*/
func getdata(client *firestore.Client, ctx context.Context, username string, userUrl string) {
	userUrl = userUrl + "/profile"
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

	lastdiff, lastdiffId := getLastDiff(client, ctx, username)
	userData := getUserData(client, ctx, username)
	if lastdiff != res && userData != res {
		if lastdiffId != "" {
			setNotApproved(client, ctx, lastdiffId)
		}
		generateAndStoreDiff(client, ctx, res, username)
	} else if userData == res {
		if lastdiffId != "" {
			setNotApproved(client, ctx, lastdiffId)
		}
	}

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

	iter := client.Collection("users").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		var userId string = doc.Ref.ID
		var userUrl string
		var username string
		if str, ok := doc.Data()["identityURL"].(string); ok {
			userUrl = str
		} else {
			continue
		}
		username = fmt.Sprint(doc.Data()["username"])
		var isServiceRunning bool
		_, err = http.Get(userUrl + "/health")
		if err != nil {
			isServiceRunning = false
		} else {
			isServiceRunning = true
		}

		logHealth(client, ctx, username, isServiceRunning)
		if !isServiceRunning {
			setIdentityStatus(client, ctx, userId, "BLOCKED", username)
			newChaincode := Chaincode{
				Username:  username,
				Timestamp: time.Now(),
			}
			client.Collection("chaincodes").Add(ctx, newChaincode)
			continue
		}

		getdata(client, ctx, username, userUrl)
	}

	defer client.Close()

	return events.APIGatewayProxyResponse{
		Body:       "Awesome, Your server health is good!!!!",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
