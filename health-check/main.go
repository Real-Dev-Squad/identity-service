package main

import (
	"context"
	"fmt"
	"identity-service/layer/utils"
	"log"
	"net/http"
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

func callProfileHealth(userUrl string) {

	defer wg.Done()

	// Skip if URL is empty
	if userUrl == "" {
		fmt.Println("Empty profile URL, skipping health check")
		return
	}

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

func (d *deps) handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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
		if str, ok := doc.Data()["profileURL"].(string); ok {
			fmt.Println(str)
			totalProfilesCalled += 1
			wg.Add(1)
			go callProfileHealth(str)
		}
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
		return
	}

	d := deps{
		client: client,
		ctx:    ctx,
	}

	lambda.Start(d.handler)
}
