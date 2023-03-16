package main

import (
	"context"
	"net/http"

	"cloud.google.com/go/firestore"
)

func getUsers(client *firestore.Client, ctx context.Context) *firestore.DocumentIterator {
	users := client.Collection("users").Where("profileStatus", "==", "VERIFIED").Documents(ctx)
	return users
}

func checkIfServiceIsRunning(url string) bool {
	request, _ := http.NewRequest(http.MethodGet, url+"health", nil)
	res, err := Client.Do(request)
	if err == nil && res.StatusCode == 200 {
		return true
	} else {
		return false
	}

}
