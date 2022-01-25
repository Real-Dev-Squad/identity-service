package main

import (
	// "errors"
	"context"
	"io/ioutil"
	"net/http"
	"os"

	"fmt"
	// "io/ioutil"
	"log"
	// "net/http"

	firebase "firebase.google.com/go"
	"github.com/aws/aws-lambda-go/events"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	memberurl                    = "https://1ngy2alfy3.execute-api.us-east-2.amazonaws.com/Prod/health"
	firestoreCredentialsLocation = "/var/task/health/firebase.json"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	firestoreCred := os.Getenv("FIRESTORE")
	fmt.Println("firetsoree", firestoreCred)

	ctx := context.Background()
	sa := option.WithCredentialsJSON([]byte(firestoreCred))
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()
	fmt.Println("client", client)
	iter := client.Collection("users").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		fmt.Println("DATA", doc.Data())
	}
	fmt.Println("firestroe", err)

	resp, err := http.Get(memberurl)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	fmt.Printf("%v", string(r))

	return events.APIGatewayProxyResponse{
		Body:       "Awesome, Your server health is good!!!!",
		StatusCode: 200,
	}, nil
}

func main() {
	// lambda.Start(handler)
	Connection()
}

func Connection() {
	ctx := context.Background()
	sa := option.WithCredentialsFile("/home/mehulkc/oss/identity-service/health/firebase.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()

	fmt.Println("client", client)
	iter := client.Collection("users").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		fmt.Println("DATA", doc.Data())
	}
	fmt.Println("firestroe", err)
	for {
		collRef, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return
		}
		fmt.Printf("Found collection with id: %s\n", collRef.ID)
	}

}
