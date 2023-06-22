package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"

	// validation packages
	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var wg sync.WaitGroup

/*
 Structures
*/
type Log struct {
	Type      string                 `firestore:"type,omitempty"`
	Timestamp time.Time              `firestore:"timestamp,omitempty"`
	Meta      map[string]interface{} `firestore:"meta,omitempty"`
	Body      map[string]interface{} `firestore:"body,omitempty"`
}

type Res struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	YOE         int    `json:"yoe"`
	Company     string `json:"company"`
	Designation string `json:"designation"`
	GithubId    string `json:"github_id"`
	LinkedIn    string `json:"linkedin_id"`
	TwitterId   string `json:"twitter_id"`
	InstagramId string `json:"instagram_id"`
	Website     string `json:"website"`
}

type Diff struct {
	UserId      string    `firestore:"userId,omitempty"`
	Timestamp   time.Time `firestore:"timestamp,omitempty"`
	Approval    string    `firestore:"approval"`
	FirstName   string    `firestore:"first_name,omitempty"`
	LastName    string    `firestore:"last_name,omitempty"`
	Email       string    `firestore:"email,omitempty"`
	Phone       string    `firestore:"phone,omitempty"`
	YOE         int       `firestore:"yoe,omitempty"`
	Company     string    `firestore:"company,omitempty"`
	Designation string    `firestore:"designation,omitempty"`
	GithubId    string    `firestore:"github_id,omitempty"`
	LinkedIn    string    `firestore:"linkedin_id,omitempty"`
	TwitterId   string    `firestore:"twitter_id,omitempty"`
	InstagramId string    `firestore:"instagram_id,omitempty"`
	Website     string    `firestore:"website,omitempty"`
}

type structProfilesSkipped struct {
	ProfileURL                         []string
	ServiceDown                        []string
	CurrentUserDataSameAsDiff          []string
	SameAsLastRejectedDiff             []string
	SameAsLastPendingDiff              []string
	ErrorInGettingProfileData          []string
	UnAuthenticatedAccessToProfileData []string
	ChaincodeNotFound                  []string
	ProfileServiceBlocked              []string
	UserDataTypeError                  []string
	ValidationError                    []string
	OtherError                         []string
}

/*
	Structures Conversions
*/
func diffToRes(diff Diff) Res {
	return Res{
		FirstName:   diff.FirstName,
		LastName:    diff.LastName,
		Email:       diff.Email,
		Phone:       diff.Phone,
		YOE:         diff.YOE,
		Company:     diff.Company,
		Designation: diff.Designation,
		GithubId:    diff.GithubId,
		LinkedIn:    diff.LinkedIn,
		TwitterId:   diff.TwitterId,
		InstagramId: diff.InstagramId,
		Website:     diff.Website,
	}
}

func resToDiff(res Res, userId string) Diff {
	return Diff{
		UserId:      userId,
		Timestamp:   time.Now(),
		Approval:    "PENDING",
		FirstName:   res.FirstName,
		LastName:    res.LastName,
		Email:       res.Email,
		Phone:       res.Phone,
		YOE:         res.YOE,
		Company:     res.Company,
		Designation: res.Designation,
		GithubId:    res.GithubId,
		LinkedIn:    res.LinkedIn,
		TwitterId:   res.TwitterId,
		InstagramId: res.InstagramId,
		Website:     res.Website,
	}
}

func diffToMap(diff Diff) map[string]interface{} {
	return map[string]interface{}{
		"userId":       diff.UserId,
		"timestamp":    diff.Timestamp,
		"approval":     diff.Approval,
		"first_name":   diff.FirstName,
		"last_name":    diff.LastName,
		"email":        diff.Email,
		"phone":        diff.Phone,
		"yoe":          diff.YOE,
		"company":      diff.Company,
		"designation":  diff.Designation,
		"github_id":    diff.GithubId,
		"linkedin_id":  diff.LinkedIn,
		"twitter_id":   diff.TwitterId,
		"instagram_id": diff.InstagramId,
		"website":      diff.Website,
	}
}

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
 Utils
*/

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

func (res Res) Validate() error {
	return validation.ValidateStruct(&res,
		validation.Field(&res.FirstName, validation.Required),
		validation.Field(&res.LastName, validation.Required),
		validation.Field(&res.Phone, validation.Required, is.Digit),
		validation.Field(&res.Email, validation.Required, is.Email),
		validation.Field(&res.YOE, validation.Min(0)),
		validation.Field(&res.Company, validation.Required),
		validation.Field(&res.Designation, validation.Required),
		validation.Field(&res.GithubId, validation.Required),
		validation.Field(&res.LinkedIn, validation.Required),
		validation.Field(&res.Website, is.URL))
}

/*
 MODELS
*/

/*
 Logs the health of the user's service
*/
func logHealth(client *firestore.Client, ctx context.Context, userId string, isServiceRunning bool) {
	newLog := Log{
		Type:      Constants["PROFILE_SERVICE_HEALTH"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: map[string]interface{}{
			"userId":         userId,
			"serviceRunning": isServiceRunning,
		},
	}
	client.Collection("logs").Add(ctx, newLog)
}

/*
 Logs the status of the user's profileDiff
*/
func logProfileSkipped(client *firestore.Client, ctx context.Context, userId string, reason string) {
	newLog := Log{
		Type:      Constants["PROFILE_SKIPPED"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: map[string]interface{}{
			"userId": userId,
			"reason": reason,
		},
	}
	client.Collection("logs").Add(ctx, newLog)
}

func logProfileStored(client *firestore.Client, ctx context.Context, userId string) {
	newLog := Log{
		Type:      Constants["PROFILE_DIFF_STORED"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: map[string]interface{}{
			"userId": userId,
		},
	}
	client.Collection("logs").Add(ctx, newLog)
}

/*
 Function for setting the profileStatus in user object in firestore
*/
func setProfileStatusBlocked(client *firestore.Client, ctx context.Context, userId string, reason string) {
	client.Collection("users").Doc(userId).Set(ctx, map[string]interface{}{
		"profileStatus": Constants["STATUS_BLOCKED"],
		"chaincode":     "",
	}, firestore.MergeAll)

	newLog := Log{
		Type:      Constants["PROFILE_SERVICE_BLOCKED"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId": userId,
		},
		Body: map[string]interface{}{
			"userId": userId,
			"reason": reason,
		},
	}
	client.Collection("logs").Add(ctx, newLog)
}

/*
 sets the user's profile diff to not approved
*/
func setNotApproved(client *firestore.Client, ctx context.Context, lastdiffId string) {
	client.Collection("profileDiffs").Doc(lastdiffId).Set(ctx, map[string]interface{}{
		"approval": Constants["NOT_APPROVED"],
	}, firestore.MergeAll)
}

/*
 Get the last profile diff of the user
*/
func getLastDiff(client *firestore.Client, ctx context.Context, userId string, approval string) (Res, string) {
	query := client.Collection("profileDiffs").Where("userId", "==", userId).Where("approval", "==", approval).OrderBy("timestamp", firestore.Desc).Limit(1).Documents(ctx)
	var lastdiff Diff
	var lastdiffId string
	for {
		Doc, err := query.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		err = Doc.DataTo(&lastdiff)
		if err != nil {
			log.Fatal(err)
		}
		lastdiffId = Doc.Ref.ID
	}
	return diffToRes(lastdiff), lastdiffId
}

/*
 Generate and Store Profile Diff
*/
func generateAndStoreDiff(client *firestore.Client, ctx context.Context, res Res, userId string) {
	var diff Diff = resToDiff(res, userId)
	_, _, err := client.Collection("profileDiffs").Add(ctx, diffToMap(diff))
	if err != nil {
		log.Fatal(err)
	} else {
		logProfileStored(client, ctx, userId)
	}
}

/*
 Getting data from the user's service
*/
func getdata(client *firestore.Client, ctx context.Context, userId string, userUrl string, chaincode string, userData Res) string {
	var status string = ""
	userUrl = userUrl + "profile"
	hashedChaincode, err := bcrypt.GenerateFromPassword([]byte(chaincode), bcrypt.DefaultCost)
	if err != nil {
		// status = Constants["SKIPPED_OTHER_ERROR"]
		// logProfileSkipped(client, ctx, userId, fmt.Sprintln(err))
		// setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err))
		return "chaincode not encrypted"
	}

	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", userUrl, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", string(hashedChaincode)))
	resp, err := httpClient.Do(req)
	if err != nil {
		// status = Constants["SKIPPED_OTHER_ERROR"]
		// logProfileSkipped(client, ctx, userId, fmt.Sprintln(err))
		// setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err))
		return "error getting profile data"
	}
	if resp.StatusCode == 401 {
		// status = Constants["PROFILE_SKIPPED_DUE_TO_UNAUTHENTICATED_ACCESS_TO_PROFILE_DATA"]
		// logProfileSkipped(client, ctx, userId, "Unauthenticated Access to Profile Data")
		// setProfileStatusBlocked(client, ctx, userId, "Unauthenticated Access to Profile Data")
		resp.Body.Close()
		return "unauthenticated access to profile data"
	}
	if resp.StatusCode != 200 {
		// status = Constants["PROFILE_SKIPPED_DUE_TO_ERROR_IN_GETTING_PROFILE_DATA"]
		// logProfileSkipped(client, ctx, userId, "Error in getting Profile Data")
		// setProfileStatusBlocked(client, ctx, userId, "Error in getting Profile Data")
		resp.Body.Close()
		return "error in getting profile data"
	}

	defer resp.Body.Close()

	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		// status = Constants["SKIPPED_OTHER_ERROR"]
		// logProfileSkipped(client, ctx, userId, fmt.Sprintln(err))
		// setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err))
		return "error reading profile data"
	}
	var res Res
	err = json.Unmarshal([]byte(r), &res)
	if err != nil {
		// status = Constants["SKIPPED_OTHER_ERROR"]
		// logProfileSkipped(client, ctx, userId, fmt.Sprintln(err))
		// setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err))
		return "error converting data to json"
	}

	err = res.Validate()

	if err != nil {
		// status = Constants["SKIPPED_VALIDATION_ERROR"]
		// logProfileSkipped(client, ctx, userId, fmt.Sprintln(err))
		// setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err))
		return fmt.Sprintf("error in validation: ", err)
	}

	lastPendingDiff, lastPendingDiffId := getLastDiff(client, ctx, userId, "PENDING")
	if lastPendingDiff != res && userData != res {
		if lastPendingDiffId != "" {
			setNotApproved(client, ctx, lastPendingDiffId)
		}
		lastRejectedDiff, lastRejectedDiffId := getLastDiff(client, ctx, userId, Constants["NOT_APPROVED"])
		if lastRejectedDiff != res {
			generateAndStoreDiff(client, ctx, res, userId)
		} else {
			status = "same last rejected diff " + lastRejectedDiffId
			// logProfileSkipped(client, ctx, userId, "Last Rejected Diff is same as New Profile Data. Rejected Diff Id: "+lastRejectedDiffId)
		}
	} else if userData == res {
		status = "same data exists"
		// logProfileSkipped(client, ctx, userId, "Current User Data is same as New Profile Data")
		if lastPendingDiffId != "" {
			setNotApproved(client, ctx, lastPendingDiffId)
		}
	} else {
		status = "same last pending diff"
		logProfileSkipped(client, ctx, userId, "Last Pending Diff is same as New Profile Data")
	}
	return status
}

/*
 Function to extract userId from the request body
*/
func getUserIdFromBody(body []byte) string {
	type extractedBody struct {
		UserId string `json:"userId"`
		// Username string `json:"username"`
		// Chaincode string `json:"chaincode"`
		// UserUrl string `json:"userUrl"`
		// ReportId string `json:"reportId"`
	}

	var e extractedBody
	json.Unmarshal(body, &e)
	return e.UserId
}

/*
 Main Handler Function
*/
func handler(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	ctx := context.Background()
	client, err := initializeFirestoreClient(ctx)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	var userId string = getUserIdFromBody([]byte(request.Body))
	if userId == "" {
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No UserID",
			StatusCode: 200,
		}, nil
	}

	dsnap, err := client.Collection("users").Doc(userId).Get(ctx)

	var userUrl string
	var chaincode string
	var username string

	if str, ok := dsnap.Data()["username"].(string); ok {
		username = str
	}

	if str, ok := dsnap.Data()["profileURL"].(string); ok {
		userUrl = str
	} else {
		// profilesSkipped.ProfileURL = append(profilesSkipped.ProfileURL, username)
		// logProfileSkipped(client, ctx, userId, "Profile URL not available")
		// setProfileStatusBlocked(client, ctx, userId, "Profile URL not available")
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No Profile URL",
			StatusCode: 200,
		}, nil
	}

	if str, ok := dsnap.Data()["chaincode"].(string); ok {
		if str == "" {
			// profilesSkipped.ProfileServiceBlocked = append(profilesSkipped.ProfileServiceBlocked, username)
			// logProfileSkipped(client, ctx, userId, "Profile Service Blocked or Chaincode is empty")
			// setProfileStatusBlocked(client, ctx, userId, "Profile Service Blocked or Chaincode is empty")
			return events.APIGatewayProxyResponse{
				Body:       "Profile Skipped Profile Service Blocked",
				StatusCode: 200,
			}, nil
		}
		chaincode = str
	} else {
		// profilesSkipped.ChaincodeNotFound = append(profilesSkipped.ChaincodeNotFound, username)
		// logProfileSkipped(client, ctx, userId, "Chaincode Not Found")
		// setProfileStatusBlocked(client, ctx, userId, "Chaincode Not Found")
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped Chaincode Not Found",
			StatusCode: 200,
		}, nil
	}

	var userData Diff
	err = dsnap.DataTo(&userData)
	if err != nil {
		// profilesSkipped.UserDataTypeError = append(profilesSkipped.UserDataTypeError, username+" Error: "+fmt.Sprintln(err))
		// logProfileSkipped(client, ctx, userId, "UserData Type Error: "+fmt.Sprintln(err))
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No User Data",
			StatusCode: 200,
		}, nil
	}

	if userUrl[len(userUrl)-1] != '/' {
		userUrl = userUrl + "/"
	}
	var isServiceRunning bool
	_, serviceErr := http.Get(userUrl + "health")
	if serviceErr != nil {
		isServiceRunning = false
	} else {
		isServiceRunning = true
	}

	// logHealth(client, ctx, userId, isServiceRunning)
	if !isServiceRunning {
		// profilesSkipped.ServiceDown = append(profilesSkipped.ServiceDown, username)
		// logProfileSkipped(client, ctx, userId, "Profile Service Down")
		// setProfileStatusBlocked(client, ctx, userId, "Profile Service Down")
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped Service Down",
			StatusCode: 200,
		}, nil
	}

	fmt.Println(userId, userUrl, chaincode, userUrl, username, isServiceRunning)

	dataErr := getdata(client, ctx, userId, userUrl, chaincode, diffToRes(userData))
	// if status == Constants["SKIPPED_SAME_LAST_PENDING_DIFF"] {
	// 	profilesSkipped.SameAsLastPendingDiff = append(profilesSkipped.SameAsLastPendingDiff, username)
	// } else if status == Constants["SKIPPED_CURRENT_USER_DATA_SAME_AS_DIFF"] {
	// 	profilesSkipped.CurrentUserDataSameAsDiff = append(profilesSkipped.CurrentUserDataSameAsDiff, username)
	// } else if status == Constants["SKIPPED_SAME_LAST_REJECTED_DIFF"] {
	// 	profilesSkipped.SameAsLastRejectedDiff = append(profilesSkipped.SameAsLastRejectedDiff, username)
	// } else if status == Constants["PROFILE_SKIPPED_DUE_TO_ERROR_IN_GETTING_PROFILE_DATA"] {
	// 	profilesSkipped.ErrorInGettingProfileData = append(profilesSkipped.ErrorInGettingProfileData, username)
	// } else if status == Constants["PROFILE_SKIPPED_DUE_TO_UNAUTHENTICATED_ACCESS_TO_PROFILE_DATA"] {
	// 	profilesSkipped.UnAuthenticatedAccessToProfileData = append(profilesSkipped.UnAuthenticatedAccessToProfileData, username)
	// } else if status == Constants["SKIPPED_OTHER_ERROR"] {
	// 	profilesSkipped.OtherError = append(profilesSkipped.OtherError, username)
	// } else if status == Constants["SKIPPED_VALIDATION_ERROR"] {
	// 	profilesSkipped.ValidationError = append(profilesSkipped.ValidationError, username)
	// } else {
	// 	*profileDiffsStored = append(*profileDiffsStored, username)
	// }
	if dataErr != "" {
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped " + dataErr,
			StatusCode: 200,
		}, nil
	}

	defer client.Close()
	return events.APIGatewayProxyResponse{
		Body:       "Profile Saved",
		StatusCode: 200,
	}, nil
}

/*
 Starts the lambda (Entry Point)
*/
func main() {
	lambda.Start(handler)
}
