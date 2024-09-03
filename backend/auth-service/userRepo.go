package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go.opentelemetry.io/otel/trace"

	// "log"
	"net/http"
	"os"
	"time"

	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// UserRepo is a repository for MongoDB operations related to User.
type UserRepo struct {
	cli                          *mongo.Client
	logger                       *log.Logger
	prof_service_string          string
	reservation_service_string   string
	accommodation_service_string string
	tracer                       trace.Tracer
}

// New creates a new UserRepo instance.
func New(ctx context.Context, logger *log.Logger, conn_address_string string, conn_reservation_service_address string,
	conn_accommodation_service_address string, tracer trace.Tracer) (*UserRepo, error) {
	dbURI := os.Getenv("MONGO_DB_URI")

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(dbURI))
	if err != nil {
		return nil, err
	}

	return &UserRepo{
		cli:                          client,
		logger:                       logger,
		prof_service_string:          conn_address_string,
		reservation_service_string:   conn_reservation_service_address,
		accommodation_service_string: conn_accommodation_service_address,
		tracer:                       tracer,
	}, nil
}

// Disconnect disconnects from the MongoDB client.
func (uh *UserRepo) Disconnect(ctx context.Context) error {
	err := uh.cli.Disconnect(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Ping checks the MongoDB connection and prints available databases.
func (uh *UserRepo) Ping() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check connection -> if no error, connection is established
	err := uh.cli.Ping(ctx, readpref.Primary())
	if err != nil {
		uh.logger.Println(err)
	}

	// Print available databases
	databases, err := uh.cli.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		uh.logger.Println(err)
	}
	fmt.Println(databases)
}

// Insert inserts a new user into the MongoDB collection.
func (uh *UserRepo) Insert(newUser *User, ctx context.Context) (*http.Response, error) {
	ctx, span := uh.tracer.Start(ctx, "UserRepo.Insert")
	defer span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	usersCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Duplicate key error: ", err)
		return nil, err
	}

	userA := uh.decodeUserA(newUser)

	result, err := usersCollection.InsertOne(ctx, userA)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	uh.logger.Printf("Document ID: %v\n", result.InsertedID)

	url := uh.prof_service_string + "/api/prof/create"

	insertedID := result.InsertedID.(primitive.ObjectID).Hex()
	log.Println("User: ", insertedID)

	userB := uh.decodeUserB(insertedID, newUser)
	reqBody, err := json.Marshal(userB)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return nil, err
	}

	log.Println("Url:", url)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := http.Client{Transport: tr}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	// httpResp, err := http.DefaultClient.Do(req)
	// if err != nil {
	// 	return nil, err
	// }
	return httpResp, nil
}

// GetAll retrieves all users from the MongoDB collection.
func (uh *UserRepo) GetAll(ctx context.Context) (Users, error) {
	ctx, span := uh.tracer.Start(ctx, "UserRepo.GetAll")
	defer span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Duplicate key error: ", err)
		return nil, err
	}

	uh.logger.Println("Collection: ", usersCollection)

	var users Users
	usersCursor, err := usersCollection.Find(ctx, bson.M{})
	if err != nil {
		log.Println("Cant find userCollection: ", err)
		return nil, err
	}
	if err = usersCursor.All(ctx, &users); err != nil {
		uh.logger.Println("User Cursor.All: ", err)
		return nil, err
	}
	return users, nil
}

func (uh *UserRepo) GetAllReservatinsDatesByHostId(hostId string, ctx context.Context) (*http.Response, error) {
	ctx, span := uh.tracer.Start(ctx, "UserRepo.Insert")
	defer span.End()

	url := uh.reservation_service_string + "/api/reservations/for_host_id/" + hostId

	log.Println("Url", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := http.Client{Transport: tr}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return httpResp, nil
}

func (uh *UserRepo) GetAllReservatinsForUser(token string, ctx context.Context) (*http.Response, error) {
	ctx, span := uh.tracer.Start(ctx, "UserRepo.GetAllReservatinsForUser")
	defer span.End()

	url := uh.reservation_service_string + "/api/reservations/by_user"

	log.Println("Url", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := http.Client{Transport: tr}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return httpResp, nil
}

// GetByUsername retrieves a user by username from the MongoDB collection.
func (uh *UserRepo) GetByUsername(username string, ctx context.Context) (*User, error) {
	ctx, span := uh.tracer.Start(ctx, "UserRepo.GetByUsername")
	defer span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Error getting collection: ", err)
		return nil, err
	}
	var user User
	log.Println("Querying for user with username: ", username)
	// objUsername, _ := primitive.ObjectIDFromHex(username)
	err = usersCollection.FindOne(ctx, bson.M{"username": username}).Decode(&user)
	if err != nil {
		log.Println("Error decoding user document: ", err)
		return nil, err
	}
	return &user, nil
}

func (uh *UserRepo) GetById(id string, ctx context.Context) (*Userr, error) {
	ctx, span := uh.tracer.Start(ctx, "UserRepo.GetById")
	defer span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Error getting collection: ", err)
		return nil, err
	}
	var user Userr
	objID, _ := primitive.ObjectIDFromHex(id)
	log.Println("Querying for user with id: ", id)
	// objUsername, _ := primitive.ObjectIDFromHex(username)
	err = usersCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		log.Println("Error decoding user document: ", err)
		return nil, err
	}
	return &user, nil
}

func (uh *UserRepo) UpdateUsersPassword(user *UserA, ctx context.Context) error { //
	ctx, span := uh.tracer.Start(ctx, "UserRepo.UpdateUsersPassword")
	defer span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	userCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Error getting collection: ", err)
		return err
	}
	filter := bson.M{"email": user.Email}
	update := bson.M{"$set": bson.M{
		"password": user.Password,
	}}
	result, err := userCollection.UpdateOne(ctx, filter, update)
	log.Printf("Documents matched: %v\n", result.MatchedCount)
	log.Printf("Documents updated: %v\n", result.ModifiedCount)

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}
func (uh *UserRepo) CreateVerificationEmail(verificationEmil VerifyEmail) error { // MORA  DA PRIMA POINTER NA VERIFYEMAIL
	//ctx, span := uh.tracer.Start(ctx, "UserRepo.Insert")
	//defer span.End()

	log.Println("Usli u CreateVerificationEmail")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	verificationCopy := verificationEmil

	verification := uh.getEmailCollection()
	result, err := verification.InsertOne(ctx, &verificationCopy)
	if err != nil {
		log.Println("Error u kreiranju verifikacionog mejla")
		return err
	}
	uh.logger.Printf("Documents ID: %v\n", result.InsertedID)
	return nil
}

func (uh *UserRepo) CreateForgottenPasswordEmail(forgottenPasswordEmail ForgottenPasswordEmail) error { // MORA  DA PRIMA POINTER NA VERIFYEMAIL
	//ctx, span := uh.tracer.Start(ctx, "UserRepo.Insert")
	//defer span.End()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	verificationCopy := forgottenPasswordEmail

	verification := uh.getFogottenPasswordEmailCollection()
	result, err := verification.InsertOne(ctx, &verificationCopy)
	if err != nil {
		uh.logger.Println(err)
		return err
	}
	uh.logger.Printf("Documents ID: %v\n", result.InsertedID)
	return nil
}

func (uh *UserRepo) GetVerificationEmailByCode(code string, ctx context.Context) (*VerifyEmail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var verifycationEmail VerifyEmail
	verifycationEmailCollection := uh.getEmailCollection()
	err := verifycationEmailCollection.FindOne(ctx, bson.M{"secretCode": code}).Decode(&verifycationEmail)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &verifycationEmail, nil
}

func (uh *UserRepo) GetForgottenPasswordEmailByCode(code string) (*ForgottenPasswordEmail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var forgottenPasswordEmail ForgottenPasswordEmail
	forgottenPasswordEmailCollection := uh.getFogottenPasswordEmailCollection() //
	err := forgottenPasswordEmailCollection.FindOne(ctx, bson.M{"secretCode": code}).Decode(&forgottenPasswordEmail)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &forgottenPasswordEmail, nil
}

func (uh *UserRepo) IsVerificationEmailActive(code string, ctx context.Context) (bool, error) {
	verificationEmail, err := uh.GetVerificationEmailByCode(code, ctx)
	if err != nil {
		return false, err
	}

	currentTime := time.Now()
	return currentTime.Before(verificationEmail.ExpiredAt), nil
}

func (uh *UserRepo) DeleteUser(username string, ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	patientsCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Error in geting collection", err)
		return err
	}
	filter := bson.M{"username": username}
	result, err := patientsCollection.DeleteOne(ctx, filter)
	if err != nil {
		log.Println(err)
		return err
	}
	log.Printf("Documents deleted: %v\n", result.DeletedCount)
	return nil
}

func (uh *UserRepo) DeleteUserInProfService(id string, ctx context.Context) (*http.Response, error) {
	url := uh.prof_service_string + "/api/prof/delete/" + id

	log.Println("Url", url)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := http.Client{Transport: tr}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return httpResp, nil
}

func (uh *UserRepo) IsForgottenPasswordEmailActive(code string, ctx context.Context) (bool, error) {
	verificationEmail, err := uh.GetForgottenPasswordEmailByCode(code)
	if err != nil {
		return false, err
	}

	currentTime := time.Now()
	return currentTime.Before(verificationEmail.ExpiredAt), nil
}

func (uh *UserRepo) GetVerificationEmailByUsername(username string, ctx context.Context) (*VerifyEmail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var verifycationEmail VerifyEmail
	verifycationEmailCollection := uh.getEmailCollection()
	err := verifycationEmailCollection.FindOne(ctx, bson.M{"username": username}).Decode(&verifycationEmail)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return &verifycationEmail, nil
}

func (uh *UserRepo) DeleteAccommdation(username string, ctx context.Context) (*http.Response, error) {
	url := uh.accommodation_service_string + "/api/accommodations/delete/" + username

	log.Println("Url", url)

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := http.Client{Transport: tr}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return httpResp, nil
}

func (uh *UserRepo) GetAllVerificationEmailsByEmail(email string, ctx context.Context) ([]VerifyEmail, error) { // needs to check if email is validate if is not than returns [] and message to user that this is not valida email
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var verificationEmails []VerifyEmail
	verificationEmailCollection := uh.getEmailCollection()

	// Use Find instead of FindOne to get multiple documents
	cursor, err := verificationEmailCollection.Find(ctx, bson.M{"email": email})
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var verificationEmail VerifyEmail
		if err := cursor.Decode(&verificationEmail); err != nil {
			log.Println(err)
			return nil, err
		}
		verificationEmails = append(verificationEmails, verificationEmail)
	}

	if err := cursor.Err(); err != nil {
		log.Println(err)
		return nil, err
	}

	return verificationEmails, nil
}

func (uh *UserRepo) UpdateUsersVerificationEmail(username string, ctx context.Context) error {
	log.Println("Usli u UpdateUsersVerificationEmail", "Username:", username)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	userCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Cant get User collection in UpdateUserVerificationEmail method")
		return err
	}

	// objID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"username": username}
	update := bson.M{"$set": bson.M{
		"isEmailVerified": true,
	}}
	result, err := userCollection.UpdateOne(ctx, filter, update)
	log.Printf("Documents matched: %v\n", result.MatchedCount)
	log.Printf("Documents updated: %v\n", result.ModifiedCount)

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (uh *UserRepo) UpdateVerificationEmail(code string, ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	emailVerificationCollection := uh.getEmailCollection()

	filter := bson.M{"secretCode": code}
	update := bson.M{"$set": bson.M{
		"isUsed": true,
	}}

	log.Printf("Updating verification email with code: %s\n", code)
	result, err := emailVerificationCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error updating verification email: %v\n", err)
		return err
	}

	log.Printf("Documents matched: %v\n", result.MatchedCount)
	log.Printf("Documents updated: %v\n", result.ModifiedCount)

	return nil
}

func (uh *UserRepo) GetByEmail(email string, ctx context.Context) (*UserA, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usersCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Error getting collection: ", err)
		return nil, err
	}
	var user UserA
	log.Println("Querying for user with email: ", email)
	// objUsername, _ := primitive.ObjectIDFromHex(username)
	err = usersCollection.FindOne(ctx, bson.M{"email": email}).Decode(&user)
	if err != nil {
		log.Println("Error decoding user document: ", err)
		return nil, err
	}
	return &user, nil
}

func (uh *UserRepo) UpdateForgottenPasswordEmail(code string, ctx context.Context) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	emailVerificationCollection := uh.getFogottenPasswordEmailCollection()

	filter := bson.M{"secretCode": code}
	update := bson.M{"$set": bson.M{
		"isUsed": true,
	}}

	log.Printf("Updating forgotten password email with code: %s\n", code)
	result, err := emailVerificationCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		log.Printf("Error updating forgotten password email: %v\n", err)
		return err
	}

	log.Printf("Documents matched: %v\n", result.MatchedCount)
	log.Printf("Documents updated: %v\n", result.ModifiedCount)

	return nil
}
func (uh *UserRepo) decodeUserA(user *User) *UserA {
	userA := UserA{
		Username:        user.Username,
		Password:        user.Password,
		Email:           user.Email,
		IsEmailVerified: false,
		Role:            user.Role,
	}
	return &userA
}

func (uh *UserRepo) decodeUserB(userId string, user *User) *UserB {
	userB := UserB{
		ID:        userId,
		Username:  user.Username,
		Role:      user.Role,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Location:  user.Location,
	}
	return &userB
}

func (uh *UserRepo) getCollection() (*mongo.Collection, error) {
	userDatabase := uh.cli.Database("mongoDemo")
	usersCollection := userDatabase.Collection("users")

	username := mongo.IndexModel{
		Keys:    bson.D{{Key: "username", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := usersCollection.Indexes().CreateOne(context.TODO(), username)
	if err != nil {
		log.Println("Error in creatingOne unique username index")
		return nil, err
	}

	email := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}

	_, err = usersCollection.Indexes().CreateOne(context.TODO(), email)
	if err != nil {
		log.Println("Error in creatingOne unique email index")
		return nil, err
	}
	return usersCollection, nil
}

func (uh *UserRepo) getEmailCollection() *mongo.Collection {
	userDatabase := uh.cli.Database("mongoDemo")
	verificationEmailCollection := userDatabase.Collection("verificatonEmails")
	return verificationEmailCollection
}

func (uh *UserRepo) getFogottenPasswordEmailCollection() *mongo.Collection {
	userDatabase := uh.cli.Database("mongoDemo")
	forgottenPasswordEmailCollection := userDatabase.Collection("forgottenPasswordEmails")
	return forgottenPasswordEmailCollection
}

func (uh *UserRepo) UpdateEmail(user *User, ctx context.Context) error {
	// treba da promeni email i da promeni da li je mejl verifikovan
	log.Println("Usli u UpdateUser", "User:", user)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	userCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Cant get User collection in UpdateUser method")
		return err
	}

	// objID, _ := primitive.ObjectIDFromHex(id)
	filter := bson.M{"_id": user.ID}
	update := bson.M{"$set": bson.M{
		"email": user.Email,
	}}
	result, err := userCollection.UpdateOne(ctx, filter, update)
	log.Printf("Documents matched: %v\n", result.MatchedCount)
	log.Printf("Documents updated: %v\n", result.ModifiedCount)

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (uh *UserRepo) UpdateProfileServiceUser(newUser *UserB, ctx context.Context) (*http.Response, error) {
	// samo salje vec promenjene podatke na prof-service
	url := uh.prof_service_string + "/api/prof/update"

	log.Println("Update User putanja: ", url)

	// insertedID := result.InsertedID.(primitive.ObjectID).Hex()
	// log.Println("User: ", insertedID)

	// userB := uh.decodeUserB(insertedID, newUser)
	reqBody, err := json.Marshal(newUser)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := http.Client{Transport: tr}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return httpResp, nil
}

func (uh *UserRepo) UpdateGrade(userId string, grade float64, ctx context.Context) error {
	log.Println("Usli u UpdateGrade", "average grade:", grade, "userId:", userId)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	userCollection, err := uh.getCollection()
	if err != nil {
		log.Println("Cant get User collection in UpdateUser method")
		return err
	}

	objID, _ := primitive.ObjectIDFromHex(userId)
	filter := bson.M{"_id": objID}
	update := bson.M{"$set": bson.M{
		"averageGrade": grade,
	}}
	result, err := userCollection.UpdateOne(ctx, filter, update)
	log.Printf("Documents matched: %v\n", result.MatchedCount)
	log.Printf("Documents updated: %v\n", result.ModifiedCount)

	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (uh *UserRepo) IsHostFeatured(id string, ctx context.Context) (*http.Response, error) {
	url := uh.reservation_service_string + "/api/reservations/host/" + id

	log.Println("Url:", url)
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	client := http.Client{Transport: tr}
	httpResp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return httpResp, nil

}
