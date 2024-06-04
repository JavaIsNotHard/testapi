package main

import (
	"bankapi/internal/data"
	"errors"
	"fmt"
	"net/http"
	"time"
)

func (app *application) healthcheck(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "status: available")
	fmt.Fprintf(w, "environment: %s\n", app.config.env)
}

func (app *application) registerUserHanlder(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.ReadJSON(w, r, &input)
	if err != nil {
		app.log.Error("hello world")
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.Users{
		Username:  input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = user.Password.Validate()
	if err != nil {
		app.failedValidationResponse(w, r, map[string]string{"error": err.Error()})
		return
	}

	err = user.Validate()
	if err != nil {
		app.failedValidationResponse(w, r, map[string]string{"error": err.Error()})
		return
	}

	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateEmail):
			app.failedValidationResponse(w, r, map[string]string{"error": "a user with the email already exists"})
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.models.Permissions.AddForUser(user.ID, "movies:read")
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.WriteJSON(w, r, Envelope{"user": user, "activation_token": token}, nil, http.StatusCreated)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		TokenString string `json:"token"`
	}

	err := app.ReadJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user, err := app.models.Users.GetForToken(data.ScopeActivation, input.TokenString)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.failedValidationResponse(w, r, map[string]string{"error": "invalid token"})
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	user.Activated = true

	err = app.models.Users.Update(user)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.models.Tokens.DeleteForUser(data.ScopeActivation, user.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}

	err = app.WriteJSON(w, r, Envelope{"user": user}, nil, http.StatusOK)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
