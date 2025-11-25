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
	"time"

	"crypto/sha512"

	"github.com/aws/aws-lambda-go/events"
)

/*
 Controller
*/
/*
 Function to verify the user
*/
func verify(ctx context.Context, profileURL string, chaincode string, salt string) (string, error) {
	type res struct {
		Hash string `json:"hash"`
	}

	postBody, _ := json.Marshal(map[string]string{
		"salt": salt,
	})

	responseBody := bytes.NewBuffer(postBody)
	
	resp, err := utils.PostWithContext(ctx, profileURL, "application/json", responseBody, 10*time.Second)
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

func handler(d *utils.Deps, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var userId string = utils.GetUserIdFromBody([]byte(request.Body))
	if userId == "" {
		return events.APIGatewayProxyResponse{}, errors.New("no userId provided")
	}

	profileURL, profileStatus, chaincode, err := utils.GetUserData(d.Client, d.Ctx, userId)
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

	status, err := verify(d.Ctx, profileURL, chaincode, salt)
	if err != nil {
		utils.LogVerification(d.Client, d.Ctx, status, profileURL, userId)
		utils.SetProfileStatus(d.Client, d.Ctx, userId, status)
		return events.APIGatewayProxyResponse{}, err
	}
	utils.LogVerification(d.Client, d.Ctx, status, profileURL, userId)
	utils.SetProfileStatus(d.Client, d.Ctx, userId, status)

	return events.APIGatewayProxyResponse{
		Body:       "Verification Process Done",
		StatusCode: 200,
	}, nil
}

func main() {
	utils.InitializeLambdaWithFirestore("verify", handler)
}
