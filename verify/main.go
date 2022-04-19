package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
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

type Chaincode struct {
	UserId    string    `firestore:"userId,omitempty"`
	Timestamp time.Time `firestore:"timestamp,omitempty"`
}

/*
 Util
*/
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
 MODEL
*/

func logVerification(client *firestore.Client, ctx context.Context, status string, profileURL string, userId string) {
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
			"reason": "chaincode not linked",
		}
		newChaincode := Chaincode{
			UserId:    userId,
			Timestamp: time.Now(),
		}
		client.Collection("chaincodes").Add(ctx, newChaincode)
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

/*
 Function for setting the profileStatus in user object in firestore
*/
func setProfileStatus(client *firestore.Client, ctx context.Context, id string, status string) error {
	_, err := client.Collection("users").Doc(id).Set(ctx, map[string]interface{}{
		"profileStatus": status,
	}, firestore.MergeAll)

	if err != nil {
		return errors.New("unable to set profile status")
	}

	return nil
}

/*
 Function to get the chaincode using userId
*/
func getChaincode(client *firestore.Client, ctx context.Context, userId string) (string, error) {
	query := client.Collection("chaincodes").Where("userId", "==", userId).OrderBy("timestamp", firestore.Desc).Limit(1).Documents(ctx)
	var chaincode string
	for {
		chaincodeDoc, err := query.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", err
		}
		chaincode = chaincodeDoc.Ref.ID
	}
	return chaincode, nil
}

/*
 Function to get the profileURL using userId
*/
func getProfileURL(client *firestore.Client, ctx context.Context, userId string) (string, string, error) {
	dsnap, err := client.Collection("users").Doc(userId).Get(ctx)
	var profileURL string
	var profileStatus string
	if err != nil {
		return "", "", err
	}
	if str, ok := dsnap.Data()["profileURL"].(string); ok {
		profileURL = str
	} else {
		return "", "", errors.New("profile url is not a string")
	}

	if str, ok := dsnap.Data()["profileStatus"].(string); ok {
		profileStatus = str
	} else {
		profileStatus = ""
	}

	return profileURL, profileStatus, nil
}

/*
 Function to extract userId from the request body
*/
func getUserIdFromBody(body []byte) string {
	type extractedBody struct {
		UserId string `json:"userId"`
	}

	var e extractedBody
	json.Unmarshal(body, &e)
	return e.UserId
}

/*
	Function to generate random string
*/
func randSalt(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

/*
 Controller
*/
/*
 Function to verify the user
*/
func verify(profileURL string, chaincode string) (string, error) {
	type res struct {
		Hash string `json:"hash"`
	}

	rand.Seed(time.Now().UnixNano())
	var salt string = randSalt(21)

	salt = "$2b$10$" + salt + "."

	postBody, _ := json.Marshal(map[string]string{
		"salt": salt,
	})

	reqBody := bytes.NewBuffer(postBody)

	resp, err := http.Post(profileURL, "application/json", reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var re res
	json.Unmarshal([]byte(r), &re)
	err = bcrypt.CompareHashAndPassword([]byte(re.Hash), []byte(chaincode))
	if err == nil {
		return "VERIFIED", nil
	} else {
		return "BLOCKED", nil
	}
}

/*
 Main Handler Function
*/
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()
	client, err := initializeFirestoreClient(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	var userId string = getUserIdFromBody([]byte(request.Body))
	if userId == "" {
		return events.APIGatewayProxyResponse{}, errors.New("no userId provided")
	}

	profileURL, profileStatus, err := getProfileURL(client, ctx, userId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if profileURL[len(profileURL)-1] == '/' {
		profileURL = profileURL + "verification"
	} else {
		profileURL = profileURL + "/verification"
	}

	if profileStatus == "VERIFIED" {
		return events.APIGatewayProxyResponse{
			Body: "Already Verified",
		}, nil
	}

	chaincode, err := getChaincode(client, ctx, userId)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	status, err := verify(profileURL, chaincode)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	logVerification(client, ctx, status, profileURL, userId)
	setProfileStatus(client, ctx, userId, status)
	defer client.Close()

	return events.APIGatewayProxyResponse{
		Body:       "Verification Process Done",
		StatusCode: 200,
	}, nil
}

/*
 Starts the lambda (Entry Point)
*/
func main() {
	lambda.Start(handler)
}
