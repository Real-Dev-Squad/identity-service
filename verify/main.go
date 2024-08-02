package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"identity-service/layer/utils"
	"io"
	"math/rand"
	"net/http"
	"time"

	"crypto/sha512"

	"cloud.google.com/go/firestore"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

/*
Structures
*/

type deps struct {
	client *firestore.Client
	ctx    context.Context
}

/*
 Controller
*/
/*
 Function to verify the user
*/
func verify(profileURL string, chaincode string, salt string) (string, error) {
	type res struct {
		Hash string `json:"hash"`
	}

	postBody, _ := json.Marshal(map[string]string{
		"salt": salt,
	})

	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(profileURL, "application/json", responseBody)
	if err != nil {
		return "BLOCKED", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "BLOCKED", err
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
	var userId string = utils.GetUserIdFromBody([]byte(request.Body))
	if userId == "" {
		return events.APIGatewayProxyResponse{}, errors.New("no userId provided")
	}

	profileURL, profileStatus, chaincode, err := utils.GetUserData(d.client, d.ctx, userId)
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
			Body:       "Already Verified",
			StatusCode: 409,
		}, nil
	}

	source := rand.NewSource(time.Now().UnixNano())
	rng := rand.New(source)

	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")
	b := make([]rune, 21)
	for i := range b {
		b[i] = letters[rng.Intn(len(letters))]
	}
	var salt string = string(b)

	status, err := verify(profileURL, chaincode, salt)
	if err != nil {
		utils.LogVerification(d.client, d.ctx, status, profileURL, userId)
		utils.SetProfileStatus(d.client, d.ctx, userId, status)
		return events.APIGatewayProxyResponse{}, err
	}
	utils.LogVerification(d.client, d.ctx, status, profileURL, userId)
	utils.SetProfileStatus(d.client, d.ctx, userId, status)

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
	client, err := utils.InitializeFirestoreClient(ctx)
	if err != nil {
		return
	}

	d := deps{
		client: client,
		ctx:    ctx,
	}

	lambda.Start(d.handler)
}
