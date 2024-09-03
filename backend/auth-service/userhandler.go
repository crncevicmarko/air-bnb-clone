package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"

	// "log"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"

	log "github.com/sirupsen/logrus"
	"github.com/thanhpk/randstr"
	"github.com/vukasinc25/fst-airbnb/mail"
	"github.com/vukasinc25/fst-airbnb/token"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserHandler handles HTTP requests related to user operations.
type UserHandler struct {
	logger   *log.Logger
	db       *UserRepo
	jwtMaker token.Maker
	tracer   trace.Tracer
}

// NewUserHandler creates a new UserHandler.
func NewUserHandler(l *log.Logger, r *UserRepo, jwtMaker token.Maker, t trace.Tracer) *UserHandler {
	return &UserHandler{l, r, jwtMaker, t}
}

func (uh *UserHandler) Auth(w http.ResponseWriter, r *http.Request) {
	//ctx, span := uh.tracer.Start(r.Context(), "UserHandler.Auth") //tracer
	//defer span.End()

	uh.logger.Println("req received")

	dec := json.NewDecoder(r.Body)

	var rt ReqToken
	err := dec.Decode(&rt)
	if err != nil {
		uh.logger.Println(err)
		uh.logger.Println("Request decode error")
	}

	uh.logger.Println(rt.Token)

	payload, err := uh.jwtMaker.VerifyToken(rt.Token)
	if err != nil {
		// If the token verification fails, return an error
		uh.logger.Println("error in token verification")
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	respBytes, err := json.Marshal(payload.ID)
	if err != nil {
		uh.logger.Println("error while creating response")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.Write(respBytes)

}

// createUser handles user creation requests.
func (uh *UserHandler) createUser(w http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.createUser") //tracer
	defer span.End()

	contentType := req.Header.Get("Content-Type")
	mediatype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		uh.logger.Println("Error cant mimi.ParseMediaType")
		sendErrorWithMessage(w, err.Error(), http.StatusBadRequest)
		return
	}

	if mediatype != "application/json" {
		err := errors.New("expect application/json Content-Type")
		sendErrorWithMessage(w, err.Error(), http.StatusUnsupportedMediaType)
		return
	}

	uh.logger.Println("Pre decodeBody")
	rt, err := decodeBody(req.Body)
	if err != nil {
		if strings.Contains(err.Error(), "Key: 'User.Username' Error:Field validation for 'Username' failed on the 'min' tag") {
			sendErrorWithMessage(w, "Username must have minimum 6 characters", http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "Key: 'User.Password' Error:Field validation for 'Password' failed on the 'password' tag") {
			sendErrorWithMessage(w, "Password must have minimum 8 characters,minimum one big letter, numbers and special characters", http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "Key: 'User.Email' Error:Field validation for 'Email' failed on the 'email' tag") {
			sendErrorWithMessage(w, "Email format is incorrect", http.StatusBadRequest)
		} else {
			sendErrorWithMessage(w, "Ovde"+err.Error(), http.StatusBadRequest)
		}
		return
	}

	rt.IsEmailVerified = false
	rt.AverageGrade = 0.0

	sanitizedUsername := sanitizeInput(rt.Username)
	sanitizedPassword := sanitizeInput(rt.Password)
	sanitizedRole := sanitizeInput(string(rt.Role))

	rt.Username = sanitizedUsername
	rt.Password = sanitizedPassword
	rt.Role = Role(sanitizedRole)

	// Fetch the blacklist
	blacklist, err := NewBlacklistFromURL()
	if err != nil {
		uh.logger.Println("Error fetching blacklist: %v\n", err)
		return
	}

	uh.logger.Println(sanitizedUsername)
	uh.logger.Println(sanitizedPassword)
	uh.logger.Println(sanitizedRole)

	// Check if the password is blacklisted
	if blacklist.IsBlacklisted(rt.Password) {
		w.WriteHeader(http.StatusBadRequest)
		uh.logger.Println("Password is not good")
		return
	}

	uh.logger.Println("Not hashed Password: %w", rt.Password)
	// Hash the password before storing
	hashedPassword, err := HashPassword(rt.Password)
	if err != nil {
		sendErrorWithMessage(w, "", http.StatusInternalServerError)
	}
	rt.Password = hashedPassword
	uh.logger.Println("Hashed Password: %w", rt.Password)

	response, err := uh.db.Insert(rt, ctx)
	if err != nil {
		if strings.Contains(err.Error(), "username") {
			sendErrorWithMessage(w, "Provide different username", http.StatusConflict)
		} else if strings.Contains(err.Error(), "email") {
			sendErrorWithMessage(w, "Provide different email", http.StatusConflict)
		}
		return
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		uh.logger.Println("Error reading response body:", err)
		sendErrorWithMessage(w, "Error reading response body", http.StatusInternalServerError)
		return
	}

	defer response.Body.Close()
	if string(responseBody) == "User created" {
		content := `
				<h1>Verify your email</h1>
				<h1>This is a verification message from AirBnb</h1>
				<h4>Use the following code: %s</h4>
				<h4><a href="localhost:4200/verify-email">Click here</a> to verify your email.</h4>`
		subject := "Verification email"
		uh.sendEmail(rt, content, subject, true, rt.Email)
		sendErrorWithMessage(w, "User cretated. Check the email for verification code", http.StatusCreated)
	} else {
		sendErrorWithMessage(w, string(responseBody), response.StatusCode)
	}
}

// getAllUsers handles requests to retrieve all users.
func (uh *UserHandler) getAllUsers(w http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.getAllUsers") //tracer
	defer span.End()

	// Retrieve all users from the database
	users, err := uh.db.GetAll(ctx)

	if err != nil {
		uh.logger.Print("Database exception: ", err)
	}

	if users == nil {
		return
	}

	// Retrieve the authorization payload from the request context
	authPayload, ok := ctx.Value(AuthorizationPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		sendErrorWithMessage(w, "Authorization payload not found", http.StatusInternalServerError)
		return
	}

	// Check user role for authorization
	if authPayload.Role == "guest" {
		sendErrorWithMessage(w, "Unauthorized access", http.StatusUnauthorized)
		return
	}

	// Convert users to JSON and send the response
	err = users.ToJSON(w)
	if err != nil {
		sendErrorWithMessage(w, "Unable to convert to JSON", http.StatusInternalServerError)
		uh.logger.Fatal("Unable to convert to JSON:", err)
		return
	}
}

func (uh *UserHandler) GetUserByUsername(w http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.GetUserIdByUsername") //tracer
	defer span.End()

	vars := mux.Vars(req)
	username := vars["username"]

	user, err := uh.db.GetByUsername(username, ctx)
	if err != nil {
		uh.logger.Println("mongo: no documents in result: no user")
		sendErrorWithMessage(w, "No such user", http.StatusBadRequest)
		return
	}

	err = user.ToJSON(w)
	if err != nil {
		sendErrorWithMessage(w, "Unable to convert to json", http.StatusInternalServerError)
		uh.logger.Fatal("Unable to convert to json :", err)
		return
	}
}

// loginUser handles user login requests.
func (uh *UserHandler) loginUser(w http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.loginUser") //tracer
	defer span.End()

	rt, err := decodeLoginBody(req.Body)
	if err != nil {
		sendErrorWithMessage(w, err.Error(), http.StatusBadRequest)
		return
	}
	username := rt.Username
	password := rt.Password
	user, err := uh.db.GetByUsername(username, ctx)
	if err != nil {
		uh.logger.Println("mongo: no documents in result: treba da se registuje neko")
		sendErrorWithMessage(w, "No such user", http.StatusBadRequest)
		return
	}

	// prooveravamo da li korisnik ima verifikovan mejl 169,170,171,172,173,174
	uh.logger.Println(user.IsEmailVerified)
	if !user.IsEmailVerified {
		sendErrorWithMessage(w, "Email is not verified", http.StatusBadRequest)
		return
	}

	if err != nil {
		uh.logger.Print("Database exception: ", err)
	}

	// If user is not found, return an error
	if user == nil {
		sendErrorWithMessage(w, "Invalid username or password", http.StatusNotFound)
		return
	}

	// Check if the provided password matches the hashed password in the database
	err = CheckHashedPassword(password, user.Password)
	if err != nil {
		sendErrorWithMessage(w, "Invalid username or password", http.StatusUnauthorized)
		return
	}

	// Generate and send JWT token as a response
	jwtToken(user, w, uh)
}

func (uh *UserHandler) sendEmail(newUser *User, contentStr string, subjectStr string, isVerificationEmail bool, email string) error { // ako isVerificationEmial is true than VrificationEmail is sending and if is false ForgottenPasswordEmial is sending
	//ctx, span := uh.tracer.Start(req.Context(), "UserHandler.Auth") //tracer
	//defer span.End()

	uh.logger.Println("SendEmail()")

	randomCode := randstr.String(20)

	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	uh.logger.Println(tlsConfig)

	err := uh.isVerificationEmail(newUser, randomCode, isVerificationEmail)
	if err != nil {
		return err
	}

	sender := mail.NewGmailSender("Air Bnb", "mobilneaplikcijesit@gmail.com", "esrqtcomedzeapdr", tlsConfig) //postavi recoveri password
	subject := subjectStr
	content := fmt.Sprintf(contentStr, randomCode)
	to := []string{email}
	attachFiles := []string{}
	uh.logger.Println("Pre SendEmail(subject, content, to, nil, nil, attachFiles)")
	err = sender.SendEmail(subject, content, to, nil, nil, attachFiles)
	if err != nil {
		// http.Error(w, err.Error(), http.StatusBadRequest)
		uh.logger.Println("Cant send email")
		return err
	}

	return nil

	// w.WriteHeader(http.StatusCreated)
	// message := "Poslat je mail na moblineaplikacijesit@gmail.com"
	// renderJSON(w, message)
}

func (uh *UserHandler) sendForgottenPasswordEmail(w http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.sendForgottenPasswordEmail") //tracer
	defer span.End()

	uh.logger.Println("Usli u sendForgottenPasswordEmail")
	vars := mux.Vars(req)
	email := vars["email"]

	allValidEmails, err := uh.db.GetAllVerificationEmailsByEmail(email, ctx) // provera ako neko probad a posalje mejl a nije registrovan
	if err != nil {
		sendErrorWithMessage(w, "Error in geting AllVerificationEmails"+err.Error(), http.StatusBadRequest)
		return
	}

	uh.logger.Println(email)
	uh.logger.Println(allValidEmails)
	if len(allValidEmails) == 0 {
		sendErrorWithMessage(w, "No valid verification emails found for the given email 1", http.StatusBadRequest)
		return
	}

	succes := false
	for _, ve := range allValidEmails { // prolazimo kroz sve emejlove koje smo dobili sa emejlom koji smo poslali da bi videli da li je mejl verifikovan ako jeste onda moze da se posalje mejl korisniku da za zaboravljenu sifru na mejl koji je poslao
		if ve.IsUsed {
			succes = true
			break
		}
	}

	if succes {
		uh.logger.Println("Usli u succes")
		content := `
				<h1>Reset Your Password</h1>
				<h1>This is a password reset message from AirBnb</h1>
				<h4>Code for password reset: %s</h4>`
		subject := "Password Reset"
		user := &User{
			Email: email,
		}
		err := uh.sendEmail(user, content, subject, false, email)
		if err != nil {
			sendErrorWithMessage(w, "Cant send email "+err.Error(), http.StatusBadRequest)
			return
		}

		sendErrorWithMessage(w, "Please check your email for the verification code", http.StatusOK)
	} else {
		sendErrorWithMessage(w, "No valid verification emails found for the given email 2", http.StatusBadRequest)
		return
	}
}
func (uh *UserHandler) isVerificationEmail(newUser *User, randomCode string, isVerificationEmail bool) error {

	uh.logger.Println("Usli u isVerificationEmail")
	if isVerificationEmail {
		verificationEmail := VerifyEmail{
			Username:   newUser.Username,
			Email:      newUser.Email,
			SecretCode: randomCode,
			IsUsed:     false,
			CreatedAt:  time.Now(),
			ExpiredAt:  time.Now().Add(15 * time.Minute), // moras da promenis da je trajanje 15 min
		}

		uh.logger.Println("Verifikacioni mejl: ", verificationEmail)

		err := uh.db.CreateVerificationEmail(verificationEmail)
		if err != nil {
			uh.logger.Println("Cant save verification email in SendEmail()method")
			return err
		}
	} else {
		forgottenPasswordEmail := ForgottenPasswordEmail{
			Email:      newUser.Email,
			SecretCode: randomCode,
			IsUsed:     false,
			CreatedAt:  time.Now(),
			ExpiredAt:  time.Now().Add(15 * time.Minute), // moras da promenis da je trajanje 15 min
		}

		uh.logger.Println("ForgottenPassword mejl: ", forgottenPasswordEmail)
		err := uh.db.CreateForgottenPasswordEmail(forgottenPasswordEmail)
		if err != nil {
			uh.logger.Println("Cant save forgotten password email in SendEmail()method")
			return err
		}
	}
	return nil
}

func (uh *UserHandler) ChangePassword(res http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.ChangePassword") //tracer
	defer span.End()

	authPayload, ok := ctx.Value(AuthorizationPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		sendErrorWithMessage(res, "Authorization payload not found", http.StatusInternalServerError)
		return
	}

	newPassword, err := decodeNewPassword(req.Body)
	if err != nil {
		if strings.Contains(err.Error(), "Key: 'NewPassword.NewPassword' Error:Field validation for 'NewPassword' failed on the 'newPassword' tag") {
			sendErrorWithMessage(res, "NewPassword must have minimum 8 characters,minimum one big letter, numbers and special characters", http.StatusBadRequest)
		} else {
			sendErrorWithMessage(res, "Cant decode body", http.StatusBadRequest)
		}
		return
	}

	user, err := uh.db.GetByUsername(authPayload.Username, ctx)
	if err != nil {
		uh.logger.Println("Error in getting user by username", err)
		sendErrorWithMessage(res, "Cant get user by username", http.StatusBadRequest)
		return
	}

	err = CheckHashedPassword(newPassword.OldPassword, user.Password)
	if err != nil {
		sendErrorWithMessage(res, "Old password dont exists", http.StatusBadRequest)
		return
	}

	if newPassword.NewPassword != newPassword.ConfirmPassword {
		sendErrorWithMessage(res, "Confirm password must match new password", http.StatusBadRequest)
		return
	}

	uh.logger.Println("Not hashed Password: %w", newPassword.NewPassword)
	// Hash the password before storing
	hashedPassword, err := HashPassword(newPassword.NewPassword)
	if err != nil {
		sendErrorWithMessage(res, "Hash: "+err.Error(), http.StatusInternalServerError)
	}

	uh.logger.Println("Hashed Password: %w", hashedPassword)

	userA := UserA{
		Email:    user.Email,
		Password: hashedPassword,
	}
	err = uh.db.UpdateUsersPassword(&userA, ctx)
	if err != nil {
		uh.logger.Println("Error in uodating password", err)
		sendErrorWithMessage(res, "Cant update password", http.StatusInternalServerError)
		return
	}

	sendErrorWithMessage(res, "Password succesfuly changed", http.StatusOK)
}

func (uh *UserHandler) changeForgottenPassword(w http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.changeForgottenPassword") //tracer
	defer span.End()

	rt, err := decodeForgottenPasswordBody(req.Body)
	if err != nil {
		if strings.Contains(err.Error(), "Key: 'ForgottenPassword.NewPassword' Error:Field validation for 'NewPassword' failed on the 'newPassword' tag") {
			sendErrorWithMessage(w, "NewPassword must have minimum 8 characters,minimum one big letter, one number and special characters", http.StatusBadRequest)
		} else if strings.Contains(err.Error(), "required") {
			sendErrorWithMessage(w, "All fealds are required", http.StatusBadRequest)
		} else {
			sendErrorWithMessage(w, "Ovde "+err.Error(), http.StatusBadRequest)
		}
		return
	}

	if rt.NewPassword != rt.ConfirmPassword {
		sendErrorWithMessage(w, "Confirm password must be same as New password", http.StatusBadRequest)
		return
	}

	forgottenPasswordEmail, err := uh.db.GetForgottenPasswordEmailByCode(rt.Code)
	if err != nil {
		uh.logger.Println("Error in getting Email by code:", err)
		sendErrorWithMessage(w, "Code is not valid", http.StatusBadRequest)
		return
	}

	if forgottenPasswordEmail != nil {
		if !forgottenPasswordEmail.IsUsed {
			isActive, err := uh.db.IsForgottenPasswordEmailActive(rt.Code, ctx)
			if err != nil {
				uh.logger.Println("Error Code is not active")
				sendErrorWithMessage(w, err.Error(), http.StatusBadRequest)
				return
			}
			if isActive {

				// verifikacija passworda treba da se radi odmah u decodBodiu

				sanitizedPassword := sanitizeInput(rt.NewPassword)

				blacklist, err := NewBlacklistFromURL()
				if err != nil {
					uh.logger.Println("Error fetching blacklist: %v\n", err)
					return
				}

				if blacklist.IsBlacklisted(rt.NewPassword) {
					uh.logger.Println("Password is too weak, blacklist")
					sendErrorWithMessage(w, "Password is too weak", http.StatusBadRequest)
					return
				}

				user := &UserA{
					Username:        "",
					Password:        sanitizedPassword,
					Email:           forgottenPasswordEmail.Email,
					IsEmailVerified: true,
				}

				hashedPassword, err := HashPassword(sanitizedPassword)
				if err != nil {
					sendErrorWithMessage(w, "", http.StatusInternalServerError)
				}

				user.Password = hashedPassword

				err = uh.db.UpdateUsersPassword(user, ctx)
				if err != nil {
					uh.logger.Println("Error when updating password")
					sendErrorWithMessage(w, "Error when updating password "+err.Error(), http.StatusBadRequest)
					return
				}

				err = uh.db.UpdateForgottenPasswordEmail(rt.Code, ctx)
				if err != nil {
					uh.logger.Println("Error in trying to update VerificationEmail")
					sendErrorWithMessage(w, "Error in trying to update VerificationEmail", http.StatusInternalServerError)
					return
				}

				sendErrorWithMessage(w, "Password succesfuly changed", http.StatusOK)

			} else {
				sendErrorWithMessage(w, "Code is not active", http.StatusBadRequest)
				return
			}
		} else {
			sendErrorWithMessage(w, "Code that has been forwarded has been used", http.StatusBadRequest)
			return
		}
	}
}

func (uh *UserHandler) UpdateUser(res http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.UpdateUser") //tracer
	defer span.End()

	authPayload, ok := ctx.Value(AuthorizationPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		sendErrorWithMessage(res, "Authorization payload not found", http.StatusInternalServerError)
		return
	}
	uh.logger.Println("AuthPayload:", authPayload)

	uh.logger.Println("Usli u UpdateUser metodu")
	user, err := decodeProfInfoBody(req.Body) // trebao bi da je citav UserB
	if err != nil {
		sendErrorWithMessage(res, "Cant decode body", http.StatusBadRequest)
		return
	}

	uh.logger.Println("Userr:", authPayload.Role)
	userRole := authPayload.Role
	var role Role
	switch userRole {
	case "HOST":
		role = "HOST"
	case "GUEST":
		role = "GUEST"
	}
	user.ID = authPayload.ID.Hex()
	user.Username = authPayload.Username
	user.Role = role

	uh.logger.Println("UserB:", user)

	// proveravamo da li postoji user sa usernejmom ako postoji
	userUsername, err := uh.db.GetByUsername(user.Username, ctx)
	if err != nil {
		sendErrorWithMessage(res, "Cant get user by email", http.StatusInternalServerError)
		return
	}

	if userUsername == nil {
		sendErrorWithMessage(res, "User with that username dont exists", http.StatusBadRequest)
		return
	}

	if userUsername.Email != user.Email {
		userId, err := primitive.ObjectIDFromHex(user.ID)
		if err != nil {
			// Handle the error, e.g., uh.logger it or return an error
			fmt.Println("Error parsing ObjectID:", err)
			return
		}

		newUser := User{
			ID:    userId,
			Email: user.Email,
		}

		uh.logger.Println("NewUser: ", newUser)
		err = uh.db.UpdateEmail(&newUser, ctx)
		if err != nil {
			sendErrorWithMessage(res, "Cant update user by email", http.StatusInternalServerError)
			return
		}
	}

	response, err := uh.db.UpdateProfileServiceUser(user, ctx)
	if err != nil {
		sendErrorWithMessage(res, "Error in updating user in prof service", http.StatusInternalServerError)
		return
	}

	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		uh.logger.Println("Error reading response body:", err)
		sendErrorWithMessage(res, "Error reading response body", http.StatusInternalServerError)
		return
	}

	uh.logger.Println("Response from prof-service:", string(responseBody))
	sendErrorWithMessage(res, string(responseBody), response.StatusCode)
}

func (uh *UserHandler) verifyEmail(w http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.Auth") //tracer
	defer span.End()

	vars := mux.Vars(req)
	code := vars["code"]

	verificationEmail, err := uh.db.GetVerificationEmailByCode(code, ctx)
	if err != nil {
		uh.logger.Println("Error in getting verificationEmail:", err)
		sendErrorWithMessage(w, "Error in getting verificationEmail", http.StatusInternalServerError)
		return
	}

	if verificationEmail != nil {
		if !verificationEmail.IsUsed {
			isActive, err := uh.db.IsVerificationEmailActive(code, ctx)
			if err != nil {
				uh.logger.Println("Error Verification code is not active")
				sendErrorWithMessage(w, err.Error(), http.StatusBadRequest)
				return
			}
			if isActive {
				err = uh.db.UpdateUsersVerificationEmail(verificationEmail.Username, ctx)
				if err != nil {
					uh.logger.Println("Error in trying to update UsersVerificationEmail")
					sendErrorWithMessage(w, "Error in trying to update UsersVerificationEmail", http.StatusInternalServerError)
					return
				}

				err = uh.db.UpdateVerificationEmail(code, ctx)
				if err != nil {
					uh.logger.Println("Error in trying to update VerificationEmail")
					sendErrorWithMessage(w, "Error in trying to update VerificationEmail", http.StatusInternalServerError)
					return
				}

				sendErrorWithMessage(w, "Your mail have been verified", http.StatusAccepted)
			} else {
				sendErrorWithMessage(w, "Code is not active", http.StatusBadRequest)
				return
			}
		} else {
			sendErrorWithMessage(w, "Code that has been forwarded has been used", http.StatusBadRequest)
			return
		}
	}

}
func jwtToken(user *User, w http.ResponseWriter, uh *UserHandler) {
	durationStr := "45m" // Should be a constant outside the function
	duration, err := time.ParseDuration(durationStr)
	if err != nil {
		uh.logger.Println("Cannot parse duration")
		return
	}

	accessToken, accessPayload, err := uh.jwtMaker.CreateToken(
		user.ID,
		user.Username,
		string(user.Role),
		duration,
	)

	if err != nil {
		sendErrorWithMessage(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rsp := LoginUserResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	}

	e := json.NewEncoder(w)
	e.Encode(rsp)
}

// decodeBody decodes the request body into a User struct.
func decodeBody(r io.Reader) (*User, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt User
	if err := dec.Decode(&rt); err != nil {
		log.Println("Decode cant be done")
		return nil, err
	}

	if err := ValidateUser(rt); err != nil {
		log.Println("User is not succesfuly validated in ValidateUser func", err)
		return nil, err
	}

	return &rt, nil
}

func decodeProfInfoBody(r io.Reader) (*UserB, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt UserB
	if err := dec.Decode(&rt); err != nil {
		log.Println("Decode cant be done")
		return nil, err
	}

	return &rt, nil
}

func decodeNewPassword(r io.Reader) (*NewPassword, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt NewPassword
	if err := dec.Decode(&rt); err != nil {
		log.Println(err)
		return nil, err
	}

	if err := ValidateNewPassword(rt); err != nil {
		log.Println("NewPasswordCredentials are not succesfuly validated in ValidateNewPassword func", err)
		return nil, err
	}

	return &rt, nil
}

func (uh *UserHandler) GetUserById(res http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.GetUserById") //tracer
	defer span.End()

	vars := mux.Vars(req)
	id := vars["id"]
	id = strings.Trim(id, "\"")

	log.Println("UserId: ", id)

	user, err := uh.db.GetById(id, ctx)
	if err != nil {
		uh.logger.Println("Error in getting user", err)
		sendErrorWithMessage(res, "No such user", http.StatusBadRequest)
		return
	}

	if user.AverageGrade >= 4.7 {
		response, err := uh.db.IsHostFeatured(id, ctx)
		if err != nil {
			uh.logger.Println("Error reading response body for IsHostFeatured is GetUserById func:", err)
			sendErrorWithMessage(res, "Error reading response body", http.StatusInternalServerError)
			return
		}

		responseBody, err := ioutil.ReadAll(response.Body)
		if err != nil {
			uh.logger.Println("Error reading response body for IsHostFeatured is GetUserById func:", err)
			sendErrorWithMessage(res, "Error reading response body", http.StatusInternalServerError)
			return
		}

		defer response.Body.Close()
		if string(responseBody) == "true" {
			featuredUser := FeaturedUser{
				Userr: Userr{
					Username:     user.Username,
					Email:        user.Email,
					AverageGrade: user.AverageGrade,
				},
				IsHostFeatured: true,
			}
			log.Println("Host1:", featuredUser)
			renderJSON(res, featuredUser)
			return
		} else {
			featuredUser := FeaturedUser{
				Userr: Userr{
					Username:     user.Username,
					Email:        user.Email,
					AverageGrade: user.AverageGrade,
				},
				IsHostFeatured: false,
			}
			log.Println("Host2:", featuredUser)
			renderJSON(res, featuredUser)
			return
			// sendErrorWithMessage(res, string(responseBody), response.StatusCode)
		}
	}

	featuredUser := FeaturedUser{
		Userr: Userr{
			Username:     user.Username,
			Email:        user.Email,
			AverageGrade: user.AverageGrade,
		},
		IsHostFeatured: false,
	}
	renderJSON(res, featuredUser)
}

func (uh *UserHandler) DeleteUser(res http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.DeleteUser") //tracer
	defer span.End()

	authPayload, ok := ctx.Value(AuthorizationPayloadKey).(*token.Payload)
	if !ok || authPayload == nil {
		sendErrorWithMessage(res, "Authorization payload not found", http.StatusInternalServerError)
		return
	}

	token, ok := ctx.Value(AccessTokenKey).(string)
	if !ok {
		sendErrorWithMessage(res, "Authorization token not found", http.StatusInternalServerError)
		return
	}

	if authPayload.Role == "GUEST" {
		response, err := uh.db.GetAllReservatinsForUser(token, ctx)
		if err != nil {
			uh.logger.Println("Error in getting reservations by user:", err)
			sendErrorWithMessage(res, "Ovde:"+err.Error(), http.StatusInternalServerError)
			return
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			uh.logger.Println("Error in reading response body")
			sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
			return
		}

		uh.logger.Println("Response", body)

		if len(body) == 0 {
			err = uh.db.DeleteUser(authPayload.Username, ctx)
			if err != nil {
				uh.logger.Println("Can't delete user", err)
				sendErrorWithMessage(res, "Can't delete user", http.StatusBadRequest)
				return
			}
			sendErrorWithMessage(res, "User successfully deleted", http.StatusOK)
			return
		}

		var reservations Reservations
		err = json.Unmarshal(body, &reservations)
		if err != nil {
			uh.logger.Println("Error in unmarshaling reservation")
			sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
			return
		}

		var isDatePassedd = false
		for _, element := range reservations {
			uh.logger.Println("Reservation:", element)
			response, err := isDatePassed(element.EndDate)
			if err != nil {
				uh.logger.Println("Error in isDatePassed:", err)
				sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
				return
			}
			isDatePassedd = response
			if !response {
				break
			}
		}

		uh.logger.Println(isDatePassedd)

		if isDatePassedd {
			responseProf, err := uh.db.DeleteUserInProfService(authPayload.ID.Hex(), ctx)
			if err != nil {
				uh.logger.Println("Can't delete user", err)
				sendErrorWithMessage(res, "Can't delete user", http.StatusBadRequest)
				return
			}

			bodyProf, err := ioutil.ReadAll(responseProf.Body)
			if err != nil {
				uh.logger.Println("Error in reading response body")
				sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
				return
			}
			if string(bodyProf) == "User succesfully deleted" {
				err = uh.db.DeleteUser(authPayload.Username, ctx)
				if err != nil {
					uh.logger.Println("Can't delete user", err)
					sendErrorWithMessage(res, "Can't delete user", http.StatusBadRequest)
					return
				}
				sendErrorWithMessage(res, "User successfully deleted", http.StatusOK)
				return
			}

			sendErrorWithMessage(res, string(bodyProf), responseProf.StatusCode)
			return
		}

		sendErrorWithMessage(res, "Cant delete user because he has active reservations", http.StatusBadRequest)
	} else {
		response, err := uh.db.GetAllReservatinsDatesByHostId(authPayload.ID.Hex(), ctx)
		if err != nil {
			uh.logger.Println("Error in getting reservations by user:", err)
			sendErrorWithMessage(res, "Ovde:"+err.Error(), http.StatusInternalServerError)
			return
		}

		body, err := ioutil.ReadAll(response.Body)
		if err != nil {
			uh.logger.Println("Error in reading response body")
			sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
			return
		}

		if strings.Contains(string(body), "There is no active reservations for accommodations of this host") {
			deletedAcco, err := uh.db.DeleteAccommdation(authPayload.Username, ctx)
			if err != nil {
				uh.logger.Println("Error when tried to delete accommodation in DeleteAccommdation", err)
				sendErrorWithMessage(res, "Error when tried to delete accommodation in DeleteAccommdation", http.StatusInternalServerError)
				return
			}

			body, err := ioutil.ReadAll(deletedAcco.Body)
			if err != nil {
				uh.logger.Println("Error in reading response body")
				sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
				return
			}

			uh.logger.Println("Body:", string(body))

			responseProf, err := uh.db.DeleteUserInProfService(authPayload.ID.Hex(), ctx)
			if err != nil {
				uh.logger.Println("Can't delete user", err)
				sendErrorWithMessage(res, "Can't delete user", http.StatusBadRequest)
				return
			}

			bodyProf, err := ioutil.ReadAll(responseProf.Body)
			if err != nil {
				uh.logger.Println("Error in reading response body")
				sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
				return
			}
			if string(bodyProf) == "User succesfully deleted" {
				err = uh.db.DeleteUser(authPayload.Username, ctx)
				if err != nil {
					uh.logger.Println("Can't delete user", err)
					sendErrorWithMessage(res, "Can't delete user", http.StatusBadRequest)
					return
				}
				sendErrorWithMessage(res, "User successfully deleted", http.StatusOK)
				return
			}

			sendErrorWithMessage(res, string(bodyProf), responseProf.StatusCode)
			return
		} else if strings.Contains(string(body), "There is active reservations for accommodations of this host") {
			sendErrorWithMessage(res, "Cant delete user because there are active reservations", http.StatusBadRequest)
			return
		} else if strings.Contains(string(body), "There is no availability dates for that accommodation") {
			deletedAcco, err := uh.db.DeleteAccommdation(authPayload.Username, ctx)
			if err != nil {
				uh.logger.Println("Error when tried to delete accommodation in DeleteAccommdation", err)
				sendErrorWithMessage(res, "Error when tried to delete accommodation in DeleteAccommdation", http.StatusInternalServerError)
				return
			}

			body, err := ioutil.ReadAll(deletedAcco.Body)
			if err != nil {
				uh.logger.Println("Error in reading response body")
				sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
				return
			}

			uh.logger.Println("Body:", string(body))

			responseProf, err := uh.db.DeleteUserInProfService(authPayload.ID.Hex(), ctx)
			if err != nil {
				uh.logger.Println("Can't delete user", err)
				sendErrorWithMessage(res, "Can't delete user", http.StatusBadRequest)
				return
			}

			bodyProf, err := ioutil.ReadAll(responseProf.Body)
			if err != nil {
				uh.logger.Println("Error in reading response body")
				sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
				return
			}
			if string(bodyProf) == "User succesfully deleted" {
				err = uh.db.DeleteUser(authPayload.Username, ctx)
				if err != nil {
					uh.logger.Println("Can't delete user", err)
					sendErrorWithMessage(res, "Can't delete user", http.StatusBadRequest)
					return
				}
				sendErrorWithMessage(res, "User successfully deleted", http.StatusOK)
				return
			}

			sendErrorWithMessage(res, string(bodyProf), responseProf.StatusCode)
			return
		} else if strings.Contains(string(body), "There is not reservations for hosts accommodations") {
			deletedAcco, err := uh.db.DeleteAccommdation(authPayload.Username, ctx)
			if err != nil {
				uh.logger.Println("Error when tried to delete accommodation in DeleteAccommdation", err)
				sendErrorWithMessage(res, "Error when tried to delete accommodation in DeleteAccommdation", http.StatusInternalServerError)
				return
			}

			body, err := ioutil.ReadAll(deletedAcco.Body)
			if err != nil {
				uh.logger.Println("Error in reading response body")
				sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
				return
			}

			uh.logger.Println("Body:", string(body))

			responseProf, err := uh.db.DeleteUserInProfService(authPayload.ID.Hex(), ctx)
			if err != nil {
				uh.logger.Println("Can't delete user", err)
				sendErrorWithMessage(res, "Can't delete user", http.StatusBadRequest)
				return
			}

			bodyProf, err := ioutil.ReadAll(responseProf.Body)
			if err != nil {
				uh.logger.Println("Error in reading response body")
				sendErrorWithMessage(res, err.Error(), http.StatusInternalServerError)
				return
			}
			if string(bodyProf) == "User succesfully deleted" {
				err = uh.db.DeleteUser(authPayload.Username, ctx)
				if err != nil {
					uh.logger.Println("Can't delete user", err)
					sendErrorWithMessage(res, "Can't delete user", http.StatusBadRequest)
					return
				}
				sendErrorWithMessage(res, "User successfully deleted", http.StatusOK)
				return
			}

			sendErrorWithMessage(res, string(bodyProf), responseProf.StatusCode)
			return
		} else {
			sendErrorWithMessage(res, string(body), http.StatusOK)
			return
		}

	}
}

func (uh *UserHandler) UpdateUserGrade(res http.ResponseWriter, req *http.Request) {
	ctx, span := uh.tracer.Start(req.Context(), "UserHandler.UpdateUserGrade") //tracer
	defer span.End()

	uh.logger.Println("Usli u UpdateGrade")
	averageGrade, err := decodeAverageGrade(req.Body)
	if err != nil {
		uh.logger.Println("Error in decoding body: ", err)
		sendErrorWithMessage1(res, "Error in decoding body", http.StatusBadRequest)
		return
	}

	log.Println("UserId", averageGrade.UserId)
	log.Println("AverageGrade:", averageGrade.AverageGrade)

	err = uh.db.UpdateGrade(averageGrade.UserId, averageGrade.AverageGrade, ctx)
	if err != nil {
		uh.logger.Println("Error in updating grade", err)
		sendErrorWithMessage1(res, "Error in updating user average grade", http.StatusInternalServerError)
		return
	}

	sendErrorWithMessage1(res, "grade updated", http.StatusOK)
}

func decodeAverageGrade(r io.Reader) (*AverageGrade, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt AverageGrade
	if err := dec.Decode(&rt); err != nil {
		log.Println(err)
		return nil, err
	}

	return &rt, nil
}

// decodeLoginBody decodes the request body into a LoginUser struct.
func decodeLoginBody(r io.Reader) (*LoginUser, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt LoginUser
	if err := dec.Decode(&rt); err != nil {
		return nil, err
	}

	return &rt, nil
}

func decodeForgottenPasswordBody(r io.Reader) (*ForgottenPassword, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()

	var rt ForgottenPassword
	if err := dec.Decode(&rt); err != nil {
		return nil, err
	}

	if err := ValidateForgottenPassword(rt); err != nil {
		log.Println("ForgottenPasswordCredentials are not succesfuly validated in ValidateForgottenPassword func")
		return nil, err
	}

	return &rt, nil
}

// sanitizeInput replaces "<" with "&lt;" to prevent potential HTML/script injection.
func sanitizeInput(input string) string {
	sanitizedInput := strings.ReplaceAll(input, "<", "&lt;")
	return sanitizedInput
}

// renderJSON writes JSON data to the response writer.
func renderJSON(w http.ResponseWriter, v interface{}) {
	js, err := json.Marshal(v)
	if err != nil {
		log.Println("Ovde")
		sendErrorWithMessage(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// ToJSON converts a Users object to JSON and writes it to the response writer.
func (u *Users) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(u)
}

func isDatePassed(dateStr time.Time) (bool, error) {
	currentDate := time.Now()
	return dateStr.Before(currentDate), nil
}

// ToJSON converts a User object to JSON and writes it to the response writer.
func (u *User) ToJSON(w io.Writer) error {
	e := json.NewEncoder(w)
	return e.Encode(u)
}

func sendErrorWithMessage(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	errorResponse := map[string]string{"message": message}
	json.NewEncoder(w).Encode(errorResponse)
}

func sendErrorWithMessage1(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(message))
	w.WriteHeader(statusCode)
}

func (uh *UserHandler) ExtractTraceInfoMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
