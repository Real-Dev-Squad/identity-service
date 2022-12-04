package main

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/firestore"
	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestHandler(t *testing.T) {

	os.Setenv("environment", "test")
	defer os.Unsetenv("environment")

	tests := []struct {
		request events.APIGatewayProxyRequest
		expect  string
		err     error
	}{
		// Format
		// {
		// 	request: events.APIGatewayProxyRequest{},
		// 	expect:  "Verification Process Done",
		// 	err:     nil,
		// },
	} 

	ctx := context.Background()
	client, err := firestore.NewClient(ctx, "test")
	if err != nil {
		return
	}

	client.Collection("users").Doc("ACD").Set(ctx, map[string]interface{}{
		"chaincode":  "ABCD",
		"profileURL": "https://identity.dev",
	})

	d := deps{
		client: client,
		ctx:    ctx,
	}

	for _, test := range tests {
		response, err := d.handler(test.request)
		assert.IsType(t, test.err, err)
		assert.Equal(t, test.expect, response)
	}

}
