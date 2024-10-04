package main

import (
	"context"
	"fmt"
	"identity-service/layer/utils"
	"log"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"google.golang.org/api/iterator"
)

var wg sync.WaitGroup

func callProfile(userId string, sessionId string) {
	log.Printf("Calling profile for user with ID: %s\n", userId)

	defer wg.Done()

	payload := utils.ProfileLambdaCallPayload{
		UserId:    userId,
		SessionID: sessionId,
	}

	err := utils.InvokeProfileLambda(payload)
	if err != nil {
		log.Println("error calling profile lambda", err)
	}
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()
	client, err := utils.InitializeFirestoreClient(ctx)

	fmt.Println("Calling profiles - entry point")

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	docRef, _, sessionIdErr := client.Collection("identitySessionIds").Add(ctx, map[string]interface{}{
		"Timestamp": time.Now(),
	})

	if sessionIdErr != nil {
		return events.APIGatewayProxyResponse{}, sessionIdErr
	}

	totalProfilesCalled := 0

	// TODO: remove this
	allUsers := client.Collection("users").Documents(ctx)
	log.Println("All users", allUsers)

	iter := client.Collection("users").Where("profileStatus", "==", "VERIFIED").Documents(ctx)
	log.Println("Iterating over users", iter)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		totalProfilesCalled += 1
		wg.Add(1)
		go callProfile(doc.Ref.ID, docRef.ID)
	}

	wg.Wait()

	defer client.Close()
	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("Total Profiles called in session is %d", totalProfilesCalled),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
