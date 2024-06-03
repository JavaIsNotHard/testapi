package main

import (
	"bankapi/internal/data"

	"github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

func (app *application) ValidateUser(user *data.Users) error {
	return validation.ValidateStruct(&user,
		validation.Field(&user.Username, validation.Required, validation.Length(10, 20)),
		validation.Field(&user.Email, validation.Required, is.Email),
		validation.Field(&user.Password, validation.Required),
	)
}
