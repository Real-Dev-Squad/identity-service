package main

import (
	"context"
	"fmt"
	"identity-service/layer/utils"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"google.golang.org/api/iterator"
)

var wg sync.WaitGroup

type deps struct {
	client *firestore.Client
	ctx    context.Context
}

func callProfile(userId string, sessionId string) {
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

func (d *deps) handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	docRef, _, sessionIdErr := d.client.Collection("identitySessionIds").Add(d.ctx, map[string]interface{}{
		"Timestamp": time.Now(),
	})

	if sessionIdErr != nil {
		return events.APIGatewayProxyResponse{}, sessionIdErr
	}

	totalProfilesCalled := 0

	iter := d.client.Collection("users").Where("profileStatus", "==", "VERIFIED").Documents(d.ctx)
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

	return events.APIGatewayProxyResponse{
		Body:       fmt.Sprintf("Total Profiles called in session is %d", totalProfilesCalled),
		StatusCode: 200,
	}, nil
}

func main() {
	ctx := context.Background()
	client, err := utils.InitializeFirestoreClient(ctx)
	if err != nil {
		log.Fatalf("Failed to initialize Firestore client: %v", err)
	}

	d := deps{
		client: client,
		ctx:    ctx,
	}

	lambda.Start(d.handler)
}
