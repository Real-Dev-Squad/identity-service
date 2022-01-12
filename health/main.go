package main

import (
	// "errors"
	"context"
	"fmt"
	// "io/ioutil"
	"log"

	// "net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	firebase "firebase.google.com/go"
	"google.golang.org/api/option"
	"google.golang.org/api/iterator"

)

var (
	memberurl = "https://1ngy2alfy3.execute-api.us-east-2.amazonaws.com/Prod/health"
)

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// resp, err := http.Get(memberurl)

	// if err != nil {
	// 	return events.APIGatewayProxyResponse{}, err
	// }

	// r, err := ioutil.ReadAll(resp.Body)
	// if err != nil {
	// 	return events.APIGatewayProxyResponse{}, err
	// }
	// fmt.Printf("%v", string(r))

	//TODO:save the response/err in firestore

	// Use a service account
	ctx := context.Background()
	sa := option.WithCredentialsFile("./firebase.json")
	app, err := firebase.NewApp(ctx, nil, sa)
	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(app)
// create data 

defer client.Close()
// [START firestore_setup_dataset_pt2]
_, _, err = client.Collection("logs").Add(ctx, map[string]interface{}{
	"first":  "Alan",
	"middle": "Mathison",
	"last":   "Turing",
	"born":   1912,
})
if err != nil {
	log.Fatalf("Failed adding aturing: %v", err)
}



	// read tdata 
	iter := client.Collection("logs").Documents(ctx)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}
		fmt.Println(doc.Data())
	}
	if err != nil {
		log.Fatalf("Failed adding alovelace: %v", err)
	}

	return events.APIGatewayProxyResponse{
		Body:       "Awesome, Your server health is good!!!!",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
