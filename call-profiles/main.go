package main

import (
	"fmt"
	"identity-service/layer/utils"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"google.golang.org/api/iterator"
)

func callProfile(userId string, sessionId string) {
	logger := utils.GetLogger()

	payload := utils.ProfileLambdaCallPayload{
		UserId:    userId,
		SessionID: sessionId,
	}

	err := utils.InvokeProfileLambda(payload)
	if err != nil {
		logger.Error("Error calling profile lambda", err, map[string]interface{}{
			"function":  "callProfile",
			"userId":    userId,
			"sessionId": sessionId,
		})
	}
}

func handler(d *utils.Deps, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	docRef, _, sessionIdErr := d.Client.Collection("identitySessionIds").Add(d.Ctx, map[string]interface{}{
		"Timestamp": time.Now(),
	})

	if sessionIdErr != nil {
		return events.APIGatewayProxyResponse{}, sessionIdErr
	}

	totalProfilesCalled := 0

	workerPool := utils.NewWorkerPool(10, 100)
	defer workerPool.Close()

	iter := d.Client.Collection("users").Where("profileStatus", "==", "VERIFIED").Documents(d.Ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return events.APIGatewayProxyResponse{
				Body:       fmt.Sprintf("Failed to iterate users: %v", err),
				StatusCode: 500,
			}, nil
		}
		
		userId := doc.Ref.ID
		sessionId := docRef.ID
		
		totalProfilesCalled += 1
		workerPool.Submit(func() {
			callProfile(userId, sessionId)
		})
	}

	workerPool.Wait()

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("Total Profiles called in session is %d", totalProfilesCalled),
		StatusCode: 200,
	}, nil
}

func main() {
	utils.InitializeLambdaWithFirestore("call-profiles", handler)
}
