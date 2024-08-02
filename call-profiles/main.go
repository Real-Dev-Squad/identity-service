package main

import (
	"bytes"
	"context"
	"fmt"
	"identity-service/layer/utils"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"google.golang.org/api/iterator"
)

var wg sync.WaitGroup

func callProfile(userId string, sessionId string) {

	defer wg.Done()

	httpClient := &http.Client{}
	jsonBody := []byte(fmt.Sprintf(`{"userId": "%s", "sessionId": "%s"}`, userId, sessionId))
	bodyReader := bytes.NewReader(jsonBody)

	requestURL := fmt.Sprintf("%s/profile", os.Getenv("baseURL"))
	req, _ := http.NewRequest(http.MethodPost, requestURL, bodyReader)
	_, err1 := httpClient.Do(req)
	if err1 != nil {
		fmt.Println("error getting profile data", err1)
	}
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()
	client, err := utils.InitializeFirestoreClient(ctx)

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

	iter := client.Collection("users").Where("profileStatus", "==", "VERIFIED").Documents(ctx)
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
