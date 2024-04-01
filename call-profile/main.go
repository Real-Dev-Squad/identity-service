package main

import (
	"bytes"
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
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/golang-jwt/jwt/v5"
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

type Claims struct {
	jwt.RegisteredClaims
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
	"ENV_DEVELOPMENT":              "DEVELOPMENT",
	"ENV_PRODUCTION":               "PRODUCTION",
	"STORED":                       "stored",
	"FIRE_STORE_CRED":              "firestoreCred",
	"DISCORD_BOT_URL":              "discordBotURL",
	"IDENTITY_SERVICE_PRIVATE_KEY": "identityServicePrivateKey",
	"PROFILE_SERVICE_HEALTH":       "PROFILE_SERVICE_HEALTH",
	"PROFILE_SKIPPED":              "PROFILE_SKIPPED",
	"PROFILE_DIFF_STORED":          "PROFILE_DIFF_STORED",
	"STATUS_BLOCKED":               "BLOCKED",
	"PROFILE_SERVICE_BLOCKED":      "PROFILE_SERVICE_BLOCKED",
	"NOT_APPROVED":                 "NOT APPROVED",
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
func getParameter(parameter string) string {
	if os.Getenv(("environment")) == Constants["ENV_DEVELOPMENT"] {
		return os.Getenv(parameter)
	} else if os.Getenv(("environment")) == Constants["ENV_PRODUCTION"] {
		var parameterName string = parameter

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
	sa := option.WithCredentialsJSON([]byte(getParameter(Constants["FIRE_STORE_CRED"])))
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
Functions to generate jwt token
*/

func generateJWTToken() string {
	signKey, errGeneratingRSAKey := jwt.ParseRSAPrivateKeyFromPEM([]byte(getParameter(Constants["IDENTITY_SERVICE_PRIVATE_KEY"])))
	if errGeneratingRSAKey != nil {
		return ""
	}
	expirationTime := time.Now().Add(1 * time.Minute)
	t := jwt.New(jwt.GetSigningMethod("RS256"))
	t.Claims = &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	tokenString, err := t.SignedString(signKey)
	if err != nil {
		return ""
	}
	return tokenString
}

/*
 MODELS
*/

/*
Logs the health of the user's service
*/
func logHealth(client *firestore.Client, ctx context.Context, userId string, isServiceRunning bool, sessionId string) {
	newLog := Log{
		Type:      Constants["PROFILE_SERVICE_HEALTH"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId":    userId,
			"sessionId": sessionId,
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
func logProfileSkipped(client *firestore.Client, ctx context.Context, userId string, reason string, sessionId string) {
	newLog := Log{
		Type:      Constants["PROFILE_SKIPPED"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId":    userId,
			"sessionId": sessionId,
		},
		Body: map[string]interface{}{
			"userId": userId,
			"reason": reason,
		},
	}
	client.Collection("logs").Add(ctx, newLog)
}

func logProfileStored(client *firestore.Client, ctx context.Context, userId string, sessionId string) {
	newLog := Log{
		Type:      Constants["PROFILE_DIFF_STORED"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId":    userId,
			"sessionId": sessionId,
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
func setProfileStatusBlocked(client *firestore.Client, ctx context.Context, userId string, reason string, sessionId string, discordId string) {
	client.Collection("users").Doc(userId).Set(ctx, map[string]interface{}{
		"profileStatus": Constants["STATUS_BLOCKED"],
		"chaincode":     "",
		"updated_at":    time.Now().UnixMilli(),
	}, firestore.MergeAll)

	if discordId != "" {
		tokenString := generateJWTToken()
		postBody, _ := json.Marshal(map[string]string{
			"userId": discordId,
			"reason": reason,
		})

		responseBody := bytes.NewBuffer(postBody)

		httpClient := &http.Client{}
		req, _ := http.NewRequest("POST", os.Getenv(Constants["DISCORD_BOT_URL"])+"/profile/blocked", responseBody)
		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", tokenString))
		httpClient.Do(req)
	}

	newLog := Log{
		Type:      Constants["PROFILE_SERVICE_BLOCKED"],
		Timestamp: time.Now(),
		Meta: map[string]interface{}{
			"userId":    userId,
			"sessionId": sessionId,
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
func generateAndStoreDiff(client *firestore.Client, ctx context.Context, res Res, userId string, sessionId string) {
	var diff Diff = resToDiff(res, userId)
	_, _, err := client.Collection("profileDiffs").Add(ctx, diffToMap(diff))
	if err != nil {
		log.Fatal(err)
	} else {
		logProfileStored(client, ctx, userId, sessionId)
	}
}

/*
Getting data from the user's service
*/
func getdata(client *firestore.Client, ctx context.Context, userId string, userUrl string, chaincode string, userData Res, sessionId string, discordId string) string {
	var status string = ""
	userUrl = userUrl + "profile"
	hashedChaincode, err := bcrypt.GenerateFromPassword([]byte(chaincode), bcrypt.DefaultCost)
	if err != nil {
		logProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return "chaincode not encrypted"
	}

	httpClient := &http.Client{}
	req, _ := http.NewRequest("GET", userUrl, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", string(hashedChaincode)))
	resp, err := httpClient.Do(req)
	if err != nil {
		logProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return "error getting profile data"
	}
	if resp.StatusCode == 401 {
		logProfileSkipped(client, ctx, userId, "Unauthenticated Access to Profile Data", sessionId)
		setProfileStatusBlocked(client, ctx, userId, "Unauthenticated Access to Profile Data", sessionId, discordId)
		resp.Body.Close()
		return "unauthenticated access to profile data"
	}
	if resp.StatusCode != 200 {
		logProfileSkipped(client, ctx, userId, "Error in getting Profile Data", sessionId)
		setProfileStatusBlocked(client, ctx, userId, "Error in getting Profile Data", sessionId, discordId)
		resp.Body.Close()
		return "error in getting profile data"
	}

	defer resp.Body.Close()

	r, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return "error reading profile data"
	}
	var res Res
	err = json.Unmarshal([]byte(r), &res)
	if err != nil {
		logProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return "error converting data to json"
	}

	err = res.Validate()

	if err != nil {
		logProfileSkipped(client, ctx, userId, fmt.Sprintln(err), sessionId)
		setProfileStatusBlocked(client, ctx, userId, fmt.Sprintln(err), sessionId, discordId)
		return fmt.Sprintf("error in validation: ", err)
	}

	lastPendingDiff, lastPendingDiffId := getLastDiff(client, ctx, userId, "PENDING")
	if lastPendingDiff != res && userData != res {
		if lastPendingDiffId != "" {
			setNotApproved(client, ctx, lastPendingDiffId)
		}
		lastRejectedDiff, lastRejectedDiffId := getLastDiff(client, ctx, userId, Constants["NOT_APPROVED"])
		if lastRejectedDiff != res {
			generateAndStoreDiff(client, ctx, res, userId, sessionId)
		} else {
			status = "same last rejected diff " + lastRejectedDiffId
			logProfileSkipped(client, ctx, userId, "Last Rejected Diff is same as New Profile Data. Rejected Diff Id: "+lastRejectedDiffId, sessionId)
		}
	} else if userData == res {
		status = "same data exists"
		logProfileSkipped(client, ctx, userId, "Current User Data is same as New Profile Data", sessionId)
		if lastPendingDiffId != "" {
			setNotApproved(client, ctx, lastPendingDiffId)
		}
	} else {
		status = "same last pending diff"
		logProfileSkipped(client, ctx, userId, "Last Pending Diff is same as New Profile Data", sessionId)
	}
	return status
}

/*
Function to extract userId from the request body
*/
func getDataFromBody(body []byte) (string, string) {
	type extractedBody struct {
		UserId    string `json:"userId"`
		SessionId string `json:"sessionId"`
	}

	var e extractedBody
	json.Unmarshal(body, &e)
	return e.UserId, e.SessionId
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

	var userId, sessionId string = getDataFromBody([]byte(request.Body))
	if userId == "" {
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No UserID",
			StatusCode: 200,
		}, nil
	}

	dsnap, err := client.Collection("users").Doc(userId).Get(ctx)

	var userUrl string
	var chaincode string
	var discordId string

	if str, ok := dsnap.Data()["discordId"].(string); ok {
		discordId = str
	} else {
		discordId = ""
	}

	if str, ok := dsnap.Data()["profileURL"].(string); ok {
		userUrl = str
	} else {
		logProfileSkipped(client, ctx, userId, "Profile URL not available", sessionId)
		setProfileStatusBlocked(client, ctx, userId, "Profile URL not available", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No Profile URL",
			StatusCode: 200,
		}, nil
	}

	if str, ok := dsnap.Data()["chaincode"].(string); ok {
		if str == "" {
			logProfileSkipped(client, ctx, userId, "Profile Service Blocked or Chaincode is empty", sessionId)
			setProfileStatusBlocked(client, ctx, userId, "Profile Service Blocked or Chaincode is empty", sessionId, discordId)
			return events.APIGatewayProxyResponse{
				Body:       "Profile Skipped Profile Service Blocked",
				StatusCode: 200,
			}, nil
		}
		chaincode = str
	} else {
		logProfileSkipped(client, ctx, userId, "Chaincode Not Found", sessionId)
		setProfileStatusBlocked(client, ctx, userId, "Chaincode Not Found", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped Chaincode Not Found",
			StatusCode: 200,
		}, nil
	}

	var userData Diff
	err = dsnap.DataTo(&userData)
	if err != nil {
		logProfileSkipped(client, ctx, userId, "UserData Type Error: "+fmt.Sprintln(err), sessionId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped No User Data",
			StatusCode: 200,
		}, nil
	}

	if userUrl[len(userUrl)-1] != '/' {
		userUrl = userUrl + "/"
	}
	var isServiceRunning bool
	c := &http.Client{
		Timeout: 5 * time.Second,
	}
	_, serviceErr := c.Get(userUrl + "health")
	if serviceErr != nil {
		isServiceRunning = false
	} else {
		isServiceRunning = true
	}

	logHealth(client, ctx, userId, isServiceRunning, sessionId)
	if !isServiceRunning {
		logProfileSkipped(client, ctx, userId, "Profile Service Down", sessionId)
		setProfileStatusBlocked(client, ctx, userId, "Profile Service Down", sessionId, discordId)
		return events.APIGatewayProxyResponse{
			Body:       "Profile Skipped Service Down",
			StatusCode: 200,
		}, nil
	}

	dataErr := getdata(client, ctx, userId, userUrl, chaincode, diffToRes(userData), sessionId, discordId)
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
