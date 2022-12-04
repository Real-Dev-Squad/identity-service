package main

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"

	"fmt"
	"google.golang.org/api/iterator"
)

func TestHandler(t *testing.T) {

	os.Setenv("environment", "test")
	defer os.Unsetenv("environment")

	var RefId string = ""

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "test")
	if err != nil {
		return
	}

	client.Collection("users").Add(ctx, map[string]interface{}{
		"chaincode":  "abcdefgh",
		"profileURL": "https://96phoonyw3.execute-api.us-east-2.amazonaws.com/Prod",
	})

	fmt.Println("All users:")
	iter := client.Collection("users").Documents(ctx)
	for {
			doc, err := iter.Next()
			if err == iterator.Done {
					break
			}
			if err != nil {
					break
			}
			RefId = doc.Ref.ID
			fmt.Println(doc.Ref.ID)
			fmt.Println(doc.Data())
	}

	tests := []struct {
		request events.APIGatewayProxyRequest
		expect  string
		err     error
	}{
		{
			// Format
			request: events.APIGatewayProxyRequest{Body: fmt.Sprintf(`{ "userId": "%s" }`, RefId)},
			expect:  "Verification Process Done",
			err:     nil,
		},
	}

	d := deps{
		client: client,
		ctx:    ctx,
	}

	for _, test := range tests {
		response, err := d.handler(test.request)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.expect, response.Body)
	}

}
