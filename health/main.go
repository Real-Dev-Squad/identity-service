package main

import (
	// "errors"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
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

type Log struct {
	Type      string    `firestore:"type,omitempty"`
	Timestamp time.Time `firestore:"timestamp,omitempty"`
	Body      string    `firestore:"body,omitempty"`
}

func getFirestoreKey() string {
	if os.Getenv(("environment")) == "DEVELOPMENT" {
		return os.Getenv("firestoreCred")
	} else if os.Getenv(("environment")) == "PRODUCTION" {
		parameterName := flag.String("", "firestoreCred", "")
		flag.Parse()

		if *parameterName == "" {
			log.Fatalf("You must supply the name of the parameter")
		}

		sess := session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))

		svc := ssm.New(sess)

		results, err := svc.GetParameter(&ssm.GetParameterInput{
			Name: parameterName,
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

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	ctx := context.Background()
	client, err := initializeFirestoreClient(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	iter := client.Collection("users").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}

		// calling user identity url
		userUrl := fmt.Sprint(doc.Data()["identityURL"])
		var isServiceRunning bool
		_, err = http.Get(userUrl + "/health")
		if err != nil {
			isServiceRunning = false
		} else {
			isServiceRunning = true
		}

		s := fmt.Sprintf("username=%v serviceRunning=%v", doc.Data()["username"], isServiceRunning)
		newLog := Log{
			Type:      "identityHealth",
			Timestamp: time.Now(),
			Body:      s,
		}
		_, _, err = client.Collection("logs").Add(ctx, newLog)
		if err != nil {
			log.Printf("An error has occurred: %s", err)
		}
	}

	defer client.Close()

	return events.APIGatewayProxyResponse{
		Body:       "Awesome, Your server health is good!!!!",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
