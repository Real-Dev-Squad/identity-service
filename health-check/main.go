package main

import (
	"context"
	"fmt"
	"identity-service/layer/utils"
	"time"

	"github.com/aws/aws-lambda-go/events"

	"google.golang.org/api/iterator"
)

func callProfileHealth(userUrl string) {
	logger := utils.GetLogger()

	// Skip if URL is empty
	if userUrl == "" {
		logger.Warn("Empty profile URL, skipping health check", map[string]interface{}{
			"function": "callProfileHealth",
		})
		return
	}

	if userUrl[len(userUrl)-1] != '/' {
		userUrl = userUrl + "/"
	}

	requestURL := fmt.Sprintf("%shealth", userUrl)
	_, err1 := utils.GetWithContext(context.Background(), requestURL, 2*time.Second)
	if err1 != nil {
		logger.WarnWithError("Service not running", err1, map[string]interface{}{
			"function":  "callProfileHealth",
			"profileURL": userUrl,
		})
	}
}

func handler(d *utils.Deps, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	totalProfilesCalled := 0

	workerPool := utils.NewWorkerPool(20, 200)
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
		
		var user utils.User
		err = doc.DataTo(&user)
		if err == nil {
			data := doc.Data()
			profileURLVal, profileURLExists := data["profileURL"]
			if profileURLExists {
				totalProfilesCalled += 1
				if profileURL, ok := profileURLVal.(string); ok && profileURL != "" {
					workerPool.Submit(func() {
						callProfileHealth(profileURL)
					})
				}
			}
		}
	}

	workerPool.Wait()

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("Total Profiles called in session is %d", totalProfilesCalled),
		StatusCode: 200,
	}, nil
}

func main() {
	utils.InitializeLambdaWithFirestore("health-check", handler)
}
