package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/Real-Dev-Squad/identity-service/layer/utils"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"google.golang.org/api/iterator"
)

var wg sync.WaitGroup

func callProfileHealth(userUrl string) {

	defer wg.Done()

	httpClient := &http.Client{
		Timeout: 2 * time.Second,
	}
	if userUrl[len(userUrl)-1] != '/' {
		userUrl = userUrl + "/"
	}

	requestURL := fmt.Sprintf("%shealth", userUrl)
	req, _ := http.NewRequest("GET", requestURL, nil)
	_, err1 := httpClient.Do(req)
	if err1 != nil {
		fmt.Println("Service not running", err1)
	}
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()
	client, err := utils.InitializeFirestoreClient(ctx)

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
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
		if str, ok := doc.Data()["profileURL"].(string); ok {
			fmt.Println(str)
			totalProfilesCalled += 1
			wg.Add(1)
			go callProfileHealth(str)
		}
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
