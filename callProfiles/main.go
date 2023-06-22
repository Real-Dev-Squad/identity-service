package main

import (
	"context"
	// "encoding/json"
	"fmt"
	// "io/ioutil"
	"log"
	// "net/http"
	"os"
	// "sync"
	// "time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	// "golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	// // validation packages
	// "github.com/go-ozzo/ozzo-validation/v4"
	// "github.com/go-ozzo/ozzo-validation/v4/is"
)

/*
 Setting Constants Map
*/
var Constants map[string]string = map[string]string{
	"ENV_DEVELOPMENT":         "DEVELOPMENT",
	"ENV_PRODUCTION":          "PRODUCTION",
	"STORED":                  "stored",
	"FIRE_STORE_CRED":         "firestoreCred",
	"PROFILE_SERVICE_HEALTH":  "PROFILE_SERVICE_HEALTH",
	"PROFILE_SKIPPED":         "PROFILE_SKIPPED",
	"PROFILE_DIFF_STORED":     "PROFILE_DIFF_STORED",
	"STATUS_BLOCKED":          "BLOCKED",
	"PROFILE_SERVICE_BLOCKED": "PROFILE_SERVICE_BLOCKED",
	"NOT_APPROVED":            "NOT APPROVED",
	"PROFILE_SKIPPED_DUE_TO_UNAUTHENTICATED_ACCESS_TO_PROFILE_DATA": "profileSkippedDueToUnAuthenticatedAccessToProfileData",
	"PROFILE_SKIPPED_DUE_TO_ERROR_IN_GETTING_PROFILE_DATA":          "profileSkippedDueToErrorInGettingProfileData",
	"SKIPPED_SAME_LAST_REJECTED_DIFF":                               "skippedSameLastRejectedDiff",
	"SKIPPED_SAME_LAST_PENDING_DIFF":                                "skippedSameLastPendingDiff",
	"SKIPPED_CURRENT_USER_DATA_SAME_AS_DIFF":                        "skippedCurrentUserDataSameAsDiff",
	"SKIPPED_OTHER_ERROR":                                           "skippedOtherError",
	"SKIPPED_VALIDATION_ERROR":                                      "validation error",
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

func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// fmt.Println("a")
	ctx := context.Background()
	// fmt.Println("a")
	client, err := initializeFirestoreClient(ctx)
	
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
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
		fmt.Println(doc)
		// go callProfileService(client, ctx, doc, &profilesSkipped, &profileDiffsStored)
	}

	defer client.Close()
	return events.APIGatewayProxyResponse{
		Body:       "reportjson",
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(handler)
}

// func getReport(totalProfilesChecked int, profileDiffsStored []string, profilesSkipped structProfilesSkipped) map[string]interface{} {
// 	var report = map[string]interface{}{
// 		"TotalProfilesChecked": totalProfilesChecked,
// 		"Stored": map[string]interface{}{
// 			"count":     len(profileDiffsStored),
// 			"usernames": profileDiffsStored,
// 		},
// 		"Skipped": map[string]interface{}{
// 			"CurrentUserDataSameAsDiff": map[string]interface{}{
// 				"count":     len(profilesSkipped.CurrentUserDataSameAsDiff),
// 				"usernames": profilesSkipped.CurrentUserDataSameAsDiff,
// 			},
// 			"SameAsLastRejectedDiff": map[string]interface{}{
// 				"count":     len(profilesSkipped.SameAsLastRejectedDiff),
// 				"usernames": profilesSkipped.SameAsLastRejectedDiff,
// 			},
// 			"NoProfileURLCount": map[string]interface{}{
// 				"count":     len(profilesSkipped.ProfileURL),
// 				"usernames": profilesSkipped.ProfileURL,
// 			},
// 			"UnauthenticatedAccessToProfileData": map[string]interface{}{
// 				"count":     len(profilesSkipped.UnAuthenticatedAccessToProfileData),
// 				"usernames": profilesSkipped.UnAuthenticatedAccessToProfileData,
// 			},
// 			"ErrorInGettingProfileData": map[string]interface{}{
// 				"count":     len(profilesSkipped.ErrorInGettingProfileData),
// 				"usernames": profilesSkipped.ErrorInGettingProfileData,
// 			},
// 			"ServiceDown": map[string]interface{}{
// 				"count":     len(profilesSkipped.ServiceDown),
// 				"usernames": profilesSkipped.ServiceDown,
// 			},
// 			"SameAsLastPendingDiff": map[string]interface{}{
// 				"count":     len(profilesSkipped.SameAsLastPendingDiff),
// 				"usernames": profilesSkipped.SameAsLastPendingDiff,
// 			},
// 			"ProfileServiceBlockedOrChaincodeEmpty": map[string]interface{}{
// 				"count":     len(profilesSkipped.ProfileServiceBlocked),
// 				"usernames": profilesSkipped.ProfileServiceBlocked,
// 			},
// 			"ChaincodeNotFound": map[string]interface{}{
// 				"count":     len(profilesSkipped.ChaincodeNotFound),
// 				"usernames": profilesSkipped.ChaincodeNotFound,
// 			},
// 			"UserDataTypeError": map[string]interface{}{
// 				"count":     len(profilesSkipped.UserDataTypeError),
// 				"usernames": profilesSkipped.UserDataTypeError,
// 			},
// 			"ValidationError": map[string]interface{}{
// 				"count":     len(profilesSkipped.ValidationError),
// 				"usernames": profilesSkipped.ValidationError,
// 			},
// 			"OtherError": map[string]interface{}{
// 				"count":     len(profilesSkipped.OtherError),
// 				"usernames": profilesSkipped.OtherError,
// 			},
// 		},
// 	}
// 	return report
// }