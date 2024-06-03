package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/julienschmidt/httprouter"
)

type Envelope map[string]any

func (app *application) ParseParams(w http.ResponseWriter, r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, err
	}

	return id, nil
}

func (app *application) WriteJSON(w http.ResponseWriter, r *http.Request, msg any, headers http.Header, status int) error {
	resp, err := json.MarshalIndent(msg, "", " ")
	if err != nil {
		return err
	}

	resp = append(resp, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content/Type", "application/json")
	w.WriteHeader(status)
	w.Write(resp)

	return nil
}

func (app *application) ReadJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	// disallow fields that do not have a correspondence to the struct tags
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)

	if err != nil {
		// syntax error in the body of the json request
		var syntaxError *json.SyntaxError
		// error if field types or struct tags do not match
		var unmarshallTypeError *json.UnmarshalTypeError
		// error if we didn't pass a pointer to the destination or some Decoding error
		var invalidUnmarshalError *json.InvalidUnmarshalError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly formed JSON (at character %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("body contains badly formed JSON")

		case errors.As(err, &unmarshallTypeError):
			if unmarshallTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type of the field %q", unmarshallTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshallTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	// check if there are more than one json body
	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}
