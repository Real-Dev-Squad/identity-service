package main

import (
	"fmt"
	"identity-service/layer/utils"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

func handler(d *utils.Deps, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var userId, sessionId string = utils.GetDataFromBody([]byte(request.Body))
	if userId == "" {
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No UserID",
			StatusCode: 200,
		}, nil
	}

	dsnap, err := d.Client.Collection("users").Doc(userId).Get(d.Ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("Error retrieving user: %v", err),
			StatusCode: 500,
		}, nil
	}

	data := dsnap.Data()
	
	var user utils.User
	err = dsnap.DataTo(&user)
	if err != nil {
		utils.LogProfileSkipped(d.Client, d.Ctx, "UserData Type Error: "+fmt.Sprintln(err), userId, sessionId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No User Data",
			StatusCode: 200,
		}, nil
	}

	discordId := user.DiscordID

	if user.ProfileURL == "" {
		utils.LogProfileSkipped(d.Client, d.Ctx, "Profile URL not available", userId, sessionId)
		utils.SetProfileStatusBlocked(d.Client, d.Ctx, userId, "Profile URL not available", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No Profile URL",
			StatusCode: 200,
		}, nil
	}

	_, chaincodeExists := data["chaincode"]
	if !chaincodeExists {
		utils.LogProfileSkipped(d.Client, d.Ctx, "Chaincode Not Found", userId, sessionId)
		utils.SetProfileStatusBlocked(d.Client, d.Ctx, userId, "Chaincode Not Found", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped Chaincode Not Found",
			StatusCode: 200,
		}, nil
	}

	if user.Chaincode == "" {
		utils.LogProfileSkipped(d.Client, d.Ctx, "Profile Service Blocked or Chaincode is empty", userId, sessionId)
		utils.SetProfileStatusBlocked(d.Client, d.Ctx, userId, "Profile Service Blocked or Chaincode is empty", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped Profile Service Blocked",
			StatusCode: 200,
		}, nil
	}

	userUrl := user.ProfileURL
	chaincode := user.Chaincode

	var userData utils.Diff
	err = dsnap.DataTo(&userData)
	if err != nil {
		utils.LogProfileSkipped(d.Client, d.Ctx, "UserData Type Error: "+fmt.Sprintln(err), userId, sessionId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No User Data",
			StatusCode: 200,
		}, nil
	}

	if userUrl[len(userUrl)-1] != '/' {
		userUrl = userUrl + "/"
	}
	
	_, serviceErr := utils.GetWithContext(d.Ctx, userUrl+"health", 5*time.Second)
	var isServiceRunning bool
	if serviceErr != nil {
		isServiceRunning = false
	} else {
		isServiceRunning = true
	}

	utils.LogHealth(d.Client, d.Ctx, userId, isServiceRunning, sessionId)
	if !isServiceRunning {
		utils.LogProfileSkipped(d.Client, d.Ctx, "Profile Service Down", userId, sessionId)
		utils.SetProfileStatusBlocked(d.Client, d.Ctx, userId, "Profile Service Down", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped Service Down",
			StatusCode: 200,
		}, nil
	}

	err = utils.Getdata(d.Client, d.Ctx, userId, userUrl, chaincode, utils.DiffToRes(userData), sessionId, discordId)
	if err != nil {
		if profileErr, ok := err.(*utils.ProfileError); ok {
			return utils.HandleProfileSkippedError(profileErr.Message), nil
		}
		return utils.HandleProfileSkippedError(err.Error()), nil
	}

	return events.APIGatewayProxyResponse{
		Body:       "Profile Saved",
		StatusCode: 200,
	}, nil
}

func main() {
	utils.InitializeLambdaWithFirestore("call-profile", handler)
}
