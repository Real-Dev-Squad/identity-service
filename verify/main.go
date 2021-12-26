package main

import (
	// "errors"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"golang.org/x/crypto/bcrypt"
)

var (
	memberurl = "https://1ngy2alfy3.execute-api.us-east-2.amazonaws.com/Prod/verify"
	chaincode = "2346"
)

type res struct {
	Hash string `json:"hash"`
}

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	postBody, _ := json.Marshal(map[string]int{
		"salt": 10,
	})

	reqBody := bytes.NewBuffer(postBody)

	resp, err := http.Post(memberurl, "application/json", reqBody)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	defer resp.Body.Close()

	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	var re res
	json.Unmarshal([]byte(r), &re)
	fmt.Printf("%v", string(r))
	//save the response/err in firestore
	err = bcrypt.CompareHashAndPassword([]byte(re.Hash), []byte(chaincode))
	if err == nil {
		return events.APIGatewayProxyResponse{
			Body:       "Matched",
			StatusCode: 200,
		}, nil
	} else {
		return events.APIGatewayProxyResponse{
			Body:       "Not Matched",
			StatusCode: 200,
		}, nil
	}
}

func main() {
	lambda.Start(handler)
}
