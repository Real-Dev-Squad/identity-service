package main

import (
	// "errors"
	"context"
	// "io/ioutil"
	"log"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"/home/mehulkc/oss/identity-service/health/models"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/gin-gonic/gin"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	memberurl = "https://1ngy2alfy3.execute-api.us-east-2.amazonaws.com/Prod/health"
	firestoreCredentialsLocation = "/home/mehulkc/oss/identity-service/firebase.json"
)

func handler(request events.APIGatewayProxyRequest c *gin.Context) (events.APIGatewayProxyResponse, error) {

	ctx := context.Background()
	sa := option.WithCredentialsFile(firestoreCredentialsLocation)
	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalln(err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()
	var newGoals []models.User
	iter := client.Collection("users").Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Failed to iterate: %v", err)
		}

		log.Print(doc.Data())

		var tempGoals models.User
		if err := doc.DataTo(&tempGoals); err != nil {
			break
		}
		newGoals = append(newGoals, tempGoals)
	}

	c.IndentedJSON(http.StatusOK, gin.H{
		"goals":   newGoals,
		"message": "Goals returned successfully!",
	})

	//TODO: 
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

	return events.APIGatewayProxyResponse{
		Body:       "Awesome, Your server health is good!!!!",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}
