package main

import (
	"context"
	"fmt"
	"identity-service/layer/utils"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type deps struct {
	client *firestore.Client
	ctx    context.Context
}

func (d *deps) handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var userId, sessionId string = utils.GetDataFromBody([]byte(request.Body))
	if userId == "" {
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No UserID",
			StatusCode: 200,
		}, nil
	}

	dsnap, err := d.client.Collection("users").Doc(userId).Get(d.ctx)

	var userUrl string
	var chaincode string
	var discordId string

	if str, ok := dsnap.Data()["discordId"].(string); ok {
		discordId = str
	} else {
		discordId = ""
	}

	if str, ok := dsnap.Data()["profileURL"].(string); ok {
		userUrl = str
	} else {
		utils.LogProfileSkipped(d.client, d.ctx, userId, "Profile URL not available", sessionId)
		utils.SetProfileStatusBlocked(d.client, d.ctx, userId, "Profile URL not available", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No Profile URL",
			StatusCode: 200,
		}, nil
	}

	if str, ok := dsnap.Data()["chaincode"].(string); ok {
		if str == "" {
			utils.LogProfileSkipped(d.client, d.ctx, userId, "Profile Service Blocked or Chaincode is empty", sessionId)
			utils.SetProfileStatusBlocked(d.client, d.ctx, userId, "Profile Service Blocked or Chaincode is empty", sessionId, discordId)
			return events.APIGatewayProxyResponse{
				Body:       "Profile Skipped Profile Service Blocked",
				StatusCode: 200,
			}, nil
		}
		chaincode = str
	} else {
		utils.LogProfileSkipped(d.client, d.ctx, userId, "Chaincode Not Found", sessionId)
		utils.SetProfileStatusBlocked(d.client, d.ctx, userId, "Chaincode Not Found", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped Chaincode Not Found",
			StatusCode: 200,
		}, nil
	}

	var userData utils.Diff
	err = dsnap.DataTo(&userData)
	if err != nil {
		utils.LogProfileSkipped(d.client, d.ctx, userId, "UserData Type Error: "+fmt.Sprintln(err), sessionId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No User Data",
			StatusCode: 200,
		}, nil
	}

	if userUrl[len(userUrl)-1] != '/' {
		userUrl = userUrl + "/"
	}
	var isServiceRunning bool
	c := &http.Client{
		Timeout: 5 * time.Second,
	}
	_, serviceErr := c.Get(userUrl + "health")
	if serviceErr != nil {
		isServiceRunning = false
	} else {
		isServiceRunning = true
	}

	utils.LogHealth(d.client, d.ctx, userId, isServiceRunning, sessionId)
	if !isServiceRunning {
		utils.LogProfileSkipped(d.client, d.ctx, userId, "Profile Service Down", sessionId)
		utils.SetProfileStatusBlocked(d.client, d.ctx, userId, "Profile Service Down", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped Service Down",
			StatusCode: 200,
		}, nil
	}

	dataErr := utils.Getdata(d.client, d.ctx, userId, userUrl, chaincode, utils.DiffToRes(userData), sessionId, discordId)
	if dataErr != "" {
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped " + dataErr,
			StatusCode: 200,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		Body:       "Profile Saved",
		StatusCode: 200,
	}, nil
}

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
