package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"net/http"
	"bytes"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"

	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var wg sync.WaitGroup

/*
 Setting Constants Map
*/
var Constants map[string]string = map[string]string{
	"ENV_DEVELOPMENT":         "DEVELOPMENT",
	"ENV_PRODUCTION":          "PRODUCTION",
	"FIRE_STORE_CRED":         "firestoreCred",
}

/*
 Setting Firestore Key for development/production
*/
func getFirestoreKey() string {
	if os.Getenv(("environment")) == Constants["ENV_DEVELOPMENT"] {
		return os.Getenv(Constants["FIRE_STORE_CRED"])
	} else if os.Getenv(("environment")) == Constants["ENV_PRODUCTION"] {
		var parameterName string = Constants["FIRE_STORE_CRED"]

		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		svc := ssm.New(sess)

		results, err := svc.GetParameter(&ssm.GetParameterInput{
			Name: &parameterName,
		})
		if err != nil {
			log.Fatalf(err.Error())
		}

		return *results.Parameter.Value
	} else {
		return ""
	}
}

/*
 Function to initialize the firestore client
*/
func initializeFirestoreClient(ctx context.Context) (*firestore.Client, error) {
	sa := option.WithCredentialsJSON([]byte(getFirestoreKey()))
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		return nil, err
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}

	return client, nil
}

func callProfile(userId string, sessionId string) {

	defer wg.Done()

	httpClient := &http.Client{}
	jsonBody := []byte(fmt.Sprintf(`{"userId": "%s", "sessionId": "%s"}`, userId, sessionId))
	bodyReader := bytes.NewReader(jsonBody)

	requestURL := fmt.Sprintf("%s/call-profile", os.Getenv("baseURL"))
	req, _ := http.NewRequest(http.MethodPost, requestURL, bodyReader)
	_, err1 := httpClient.Do(req)
	if err1 != nil {
		fmt.Println("error getting profile data", err1)
	}
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()
	client, err := initializeFirestoreClient(ctx)

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	docRef ,_ ,sessionIdErr := client.Collection("identitySessionIds").Add(ctx, map[string]interface{}{
		"Timestamp": time.Now(),
	})

	if(sessionIdErr != nil) {
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