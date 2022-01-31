package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

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
 Function for setting the identityStatus in user object in firestore
*/
func setIdentityStatus(client *firestore.Client, ctx context.Context, id string, status string) error {
	_, err := client.Collection("users").Doc(id).Set(ctx, map[string]interface{}{
		"identityStatus": status,
	}, firestore.MergeAll)

	if err != nil {
		return errors.New("unable to set identity status")
	}

	return nil
}

/*
 Function to extract username for the request body
*/
func getUsernameFromBody(body []byte) string {
	type extractedBody struct {
		Username string `json:"username"`
	}

	var e extractedBody
	json.Unmarshal(body, &e)
	return e.Username
}

/*
 Function to get the chaincode using username
*/
func getChaincode(client *firestore.Client, ctx context.Context, username string) (string, error) {
	query := client.Collection("chaincodes").Where("username", "==", username).OrderBy("timestamp", firestore.Desc).Limit(1).Documents(ctx)
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
		fmt.Println(chaincodeDoc.Ref.ID)
	}
	return chaincode, nil
}

/*
 Function to get the identityURL using username
*/
func getIdentityURL(client *firestore.Client, ctx context.Context, username string) (string, string, string, error) {
	query := client.Collection("users").Where("username", "==", username).Limit(1).Documents(ctx)
	var identityURL string
	var identityStatus string
	var userID string
	for {
		userDoc, err := query.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return "", "", "", err
		}
		if str, ok := userDoc.Data()["identityURL"].(string); ok {
			identityURL = str
		} else {
			return "", "", "", errors.New("identity url is not a string")
		}

		if str, ok := userDoc.Data()["identityStatus"].(string); ok {
			identityStatus = str
		} else {
			identityStatus = ""
		}

		userID = userDoc.Ref.ID
	}
	return identityURL, userID, identityStatus, nil
}

/*
 Function to verify the user
*/
func verify(identityURL string, chaincode string) (string, error) {
	type res struct {
		Hash string `json:"hash"`
	}

	postBody, _ := json.Marshal(map[string]int{
		"salt": 10,
	})

	reqBody := bytes.NewBuffer(postBody)

	resp, err := http.Post(identityURL, "application/json", reqBody)
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

	var username string = getUsernameFromBody([]byte(request.Body))
	if username == "" {
		return events.APIGatewayProxyResponse{}, errors.New("no username provided")
	}

	identityURL, userId, identityStatus, err := getIdentityURL(client, ctx, username)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	identityURL = identityURL + "/verify"

	if identityStatus == "VERIFIED" {
		return events.APIGatewayProxyResponse{
			Body: "Already Verified",
		}, nil
	}

	chaincode, err := getChaincode(client, ctx, username)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	status, err := verify(identityURL, chaincode)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	setIdentityStatus(client, ctx, userId, status)
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
