package utils

import (
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/go-ozzo/ozzo-validation/is"
)

// Validation Functions

func resToDiff(res Res, userId string) Diff {
	return Diff{
		UserId:      userId,
		Timestamp:   time.Now(),
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

func DiffToRes(diff Diff) Res {
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
