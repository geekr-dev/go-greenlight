package main

import (
	"net/http"
	"time"

	"greenlight.geekr.dev/internal/data"
	"greenlight.geekr.dev/internal/data/validator"
)

func (app *application) registerUserHanlder(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	user := &data.User{
		Name:      input.Name,
		Email:     input.Email,
		Activated: false,
	}

	err = user.Password.Set(input.Password)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	v := validator.New()

	if data.ValidateUser(v, user); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Insert the user into the database.
	err = app.models.Users.Insert(user)
	if err != nil {
		switch {
		case err == data.ErrDuplicateEmail:
			v.AddError("email", "a user with this email address already exists")
			app.failedValidationResponse(w, r, v.Errors)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	token, err := app.models.Tokens.New(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Send the user a welcome email.
	app.background(func() {
		err = app.mailer.Send(user.Email, "user_welcome.tmpl", map[string]any{
			"userID":          user.ID,
			"activationToken": token.Plaintext,
		})
		if err != nil {
			app.logger.Error(err, nil)
		}
	})

	err = app.writeJSON(w, http.StatusCreated, envelope{"user": user}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
