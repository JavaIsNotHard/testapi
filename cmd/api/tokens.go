package main

import (
	"bankapi/internal/data"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt"
)

func (app *application) createAuthenticationToken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.ReadJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	existingUser, err := app.models.Users.GetUserByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialResponse(w, r)
			return
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	matches, err := existingUser.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !matches {
		app.invalidCredentialResponse(w, r)
		return
	}

	token, err := app.models.Tokens.New(existingUser.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.WriteJSON(w, r, map[string]any{"authentication_token": token}, nil, http.StatusCreated)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createJWTtoken(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.ReadJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	existingUser, err := app.models.Users.GetUserByEmail(input.Email)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.invalidCredentialResponse(w, r)
			return
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	matches, err := existingUser.Password.Matches(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	if !matches {
		app.invalidCredentialResponse(w, r)
		return
	}

	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Subject:   strconv.FormatInt(existingUser.ID, 10),
		Issuer:    "jibeshshrestha",
		IssuedAt:  jwt.TimeFunc().Unix(),
		NotBefore: jwt.TimeFunc().Unix(),
		ExpiresAt: jwt.TimeFunc().Add(24 * time.Hour).Unix(),
		Audience:  "jibeshshrestha",
	})

	token, err := claims.SignedString([]byte(SecretKey))

	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.WriteJSON(w, r, map[string]any{"authentication_token": token}, nil, http.StatusCreated)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
