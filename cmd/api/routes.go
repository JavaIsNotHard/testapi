package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	mux := httprouter.New()

	mux.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.requiredActivatedUser(app.healthcheck))
	mux.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHanlder)
	mux.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationToken)
	mux.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)

	return app.logRequest(app.authenicate(mux))
}
