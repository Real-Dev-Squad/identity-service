package main

import (
	"fmt"
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
	"crypto/sha512"
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

type deps struct {
	client *firestore.Client
	ctx    context.Context
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
	var firestoreKey = getFirestoreKey()
	if firestoreKey == "" {
		return nil, errors.New("no firestore key found")
	}
	sa := option.WithCredentialsJSON([]byte(firestoreKey))
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

/*
Function for setting the profileStatus in user object in firestore
*/
func setProfileStatus(client *firestore.Client, ctx context.Context, id string, status string) error {
	var newData = map[string]interface{}{
		"profileStatus": status,
	}

	if status == "BLOCKED" {
		newData = map[string]interface{}{
			"profileStatus": status,
			"chaincode":     "",
		}
	}

	_, err := client.Collection("users").Doc(id).Set(ctx, newData, firestore.MergeAll)

	if err != nil {
		return errors.New("unable to set profile status")
	}

	return nil
}

/*
Function to get the userData using userId
*/
func getUserData(client *firestore.Client, ctx context.Context, userId string) (string, string, string, error) {
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

	postBody, _ := json.Marshal(map[string]string{
		"salt": salt,
	})

	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(profileURL, "application/json", responseBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var re res
	json.Unmarshal([]byte(body), &re)
	sha_512 := sha512.New()
	sha_512.Write([]byte(salt + chaincode))
	if fmt.Sprintf("%x", sha_512.Sum(nil)) == re.Hash {
		return "VERIFIED", nil
	} else {
		return "BLOCKED", nil
	}
}

/*
Main Handler Function
*/
func (d *deps) handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var userId string = getUserIdFromBody([]byte(request.Body))
	if userId == "" {
		return events.APIGatewayProxyResponse{}, errors.New("no userId provided")
	}

	profileURL, profileStatus, chaincode, err := getUserData(d.client, d.ctx, userId)
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

	status, err := verify(profileURL, chaincode)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	logVerification(d.client, d.ctx, status, profileURL, userId)
	setProfileStatus(d.client, d.ctx, userId, status)

	return events.APIGatewayProxyResponse{
		Body:       "Verification Process Done",
		StatusCode: 200,
	}, nil
}

/*
Starts the lambda (Entry Point)
*/
func main() {
	ctx := context.Background()
	client, err := initializeFirestoreClient(ctx)
	if err != nil {
		return
	}

	d := deps{
		client: client,
		ctx:    ctx,
	}

	lambda.Start(d.handler)
}
