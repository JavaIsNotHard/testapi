package main

import (
	"bankapi/internal/data"
	"context"
	"net/http"
)

type contextKey string

const userContextKey = contextKey("user")

func (app *application) contextSetUser(r *http.Request, data *data.Users) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, data)
	return r.WithContext(ctx)
}

func (app *application) contextGetUser(r *http.Request) *data.Users {
	user, ok := r.Context().Value(userContextKey).(*data.Users)
	if !ok {
		panic("missing user context in the request")
	}

	return user
}
