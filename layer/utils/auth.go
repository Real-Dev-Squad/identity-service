package utils

import (
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/golang-jwt/jwt/v5"
)

// JWT and HTTP Functions

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
			log.Print(err.Error())
		}

		return *results.Parameter.Value
	} else {
		return ""
	}
}
