package main

import (
	"bankapi/internal/data"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

func (app *application) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		app.log.Info("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI())

		next.ServeHTTP(w, r)
	})
}

func (app *application) authenicate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")
		authorizationHeader := r.Header.Get("Authorization")

		if authorizationHeader == "" {
			fmt.Println("No Header found")
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		headerPart := strings.Split(authorizationHeader, " ")
		if len(headerPart) != 2 || headerPart[0] != "Bearer" {
			app.invalidCredentialResponse(w, r)
			return
		}

		authenticationtoken := headerPart[1]
		token := &data.Token{
			Token: authenticationtoken,
		}

		if err := token.Validate(); err != nil {
			app.invalidCredentialResponse(w, r)
			return
		}

		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, authenticationtoken)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationToken(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}

		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

func (app *application) requiredAuthenicatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if user.IsAnonymous() {
			app.authenticationRequiredResposne(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// automatically calls the requiredAuthenticationUser before being executed itself
func (app *application) requiredActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if !user.Activated {
			app.inactiveResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return app.requiredAuthenicatedUser(fn)
}

func (app *application) requirePermission(code string, next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)

		permissions, err := app.models.Permissions.GetAllForUser(user.ID)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}

		if !permissions.Include(code) {
			app.notPermittedResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})

	return app.requiredActivatedUser(fn)
}
