package main

import (
	"bytes"
	"crypto/tls"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go.opentelemetry.io/otel/trace"
	"net/http"
	"strings"

	"github.com/hashicorp/consul/api"
	log "github.com/sirupsen/logrus"
)

type UserRepo struct {
	cli                        *api.Client
	logger                     *log.Logger
	reservation_service_string string
	auth_service_string        string
	tracer                     trace.Tracer
}

const (
	users          = "users/%s"
	all            = "users"
	hostGrades     = "hostGrades/%s"
	hostGradeIndex = "hostGradeIndex/%s"
)

func generateKey(Id string) string {
	id := Id
	return fmt.Sprintf(users, id)
}

func New(logger *log.Logger, conn_reservation_service_address string,
	conn_auth_service_address string, tracer trace.Tracer) (*UserRepo, error) {
	config := api.DefaultConfig()
	config.Address = fmt.Sprintf("%s:%s", "consul", "8500")
	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &UserRepo{
		cli:                        client,
		logger:                     logger,
		reservation_service_string: conn_reservation_service_address,
		auth_service_string:        conn_auth_service_address,
		tracer:                     tracer,
	}, nil
}

func (ur *UserRepo) Insert(user *User, ctx context.Context) error {
	ctx, span := ur.tracer.Start(ctx, "userRepo.Insert")
	defer span.End()

	log.Println("Usli u Insert")
	kv := ur.cli.KV()

	dbId := generateKey(user.ID)

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	userKeyValue := &api.KVPair{Key: dbId, Value: data}
	_, err = kv.Put(userKeyValue, nil)
	if err != nil {
		return err
	}

	return nil
}

func (pr *UserRepo) GetAll(ctx context.Context) (Users, error) {
	ctx, span := pr.tracer.Start(ctx, "userRepo.GetAll")
	defer span.End()

	kv := pr.cli.KV()
	data, _, err := kv.List(all, nil)
	if err != nil {
		return nil, err
	}

	users := Users{}
	for _, pair := range data {
		user := &User{}
		err = json.Unmarshal(pair.Value, user)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	return users, nil
}

func (ur *UserRepo) Get(id string, ctx context.Context) (*User, error) {
	ctx, span := ur.tracer.Start(ctx, "userRepo.Get")
	defer span.End()

	kv := ur.cli.KV()

	pair, _, err := kv.Get(constructKey(id), nil)
	if err != nil {
		return nil, err
	}
	if pair == nil {
		log.Println("blabla:", pair)
		return nil, nil
	}

	user := &User{}
	err = json.Unmarshal(pair.Value, user)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func constructKey(id string) string {
	return fmt.Sprintf(users, id)
}

func (ur *UserRepo) Delete(id string, ctx context.Context) error {
	ctx, span := ur.tracer.Start(ctx, "userRepo.Delete")
	defer span.End()

	kv := ur.cli.KV()

	_, err := kv.Delete(constructKey(id), nil)
	if err != nil {
		return err
	}

	return nil
}

func (ur *UserRepo) UpdateUser(user *User, ctx context.Context) error {
	ctx, span := ur.tracer.Start(ctx, "userRepo.UpdateUser")
	defer span.End()

	kv := ur.cli.KV()

	data, err := json.Marshal(user)
	if err != nil {
		return err
	}

	productKeyValue := &api.KVPair{Key: constructKey(user.ID), Value: data}
	_, err = kv.Put(productKeyValue, nil)
	if err != nil {
		return err
	}

	return nil
}

func (ur *UserRepo) CreateHostGrade(hostGrade *HostGrade, ctx context.Context) error {
	ctx, span := ur.tracer.Start(ctx, "userRepo.CreateHostGrade")
	defer span.End()

	log.Println("Usli u CreateHostGrade")
	log.Println("HostGrade:", hostGrade)
	kv := ur.cli.KV()

	dbId := fmt.Sprintf(hostGrades, hostGrade.ID)

	data, err := json.Marshal(hostGrade)
	if err != nil {
		return err
	}

	log.Println("Data:", data)

	KeyValue := &api.KVPair{Key: dbId, Value: data}
	_, err = kv.Put(KeyValue, nil)
	if err != nil {
		return err
	}

	// Add the HostGrade ID to the index
	indexKey := fmt.Sprintf(hostGradeIndex, hostGrade.HostId)
	indexValue, _, err := kv.Get(indexKey, nil)
	if err != nil {
		return err
	}

	log.Println("Index Value:", indexValue)

	var ids []string
	if indexValue != nil {
		if err := json.Unmarshal(indexValue.Value, &ids); err != nil {
			return err
		}
	}

	// Add the new ID to the list
	ids = append(ids, hostGrade.ID)

	// Marshal the list of IDs into a JSON array
	jsonArray, err := json.Marshal(ids)
	if err != nil {
		return err
	}

	indexValue = &api.KVPair{Key: indexKey, Value: jsonArray}
	_, err = kv.Put(indexValue, nil)
	if err != nil {
		return err
	}

	return nil
}

func (ur *UserRepo) DeleteHostGrade(id string, userId string, ctx context.Context) error {
	ctx, span := ur.tracer.Start(ctx, "userRepo.DeleteHostGrade")
	defer span.End()

	kv := ur.cli.KV()

	log.Println("Ovde1")
	// Retrieve HostGrade by ID to get the HostId
	hostGradeKey := fmt.Sprintf(hostGrades, strings.TrimSpace(id))
	log.Println("HostGradeKey:", hostGradeKey)
	pair, _, err := kv.Get(hostGradeKey, nil)
	if err != nil {
		log.Println("Error in getting host grade by id:", err)
		return err
	}
	log.Println("Ovde2")

	if pair == nil {
		return errors.New("HostGrade not found")
	}

	var hg HostGrade
	if err := json.Unmarshal(pair.Value, &hg); err != nil {
		return err
	}

	if hg.UserId != strings.Trim(userId, `"`) {
		return errors.New("cant delete host grade")
	}

	log.Println("Ovde4")

	// Log initial values
	log.Printf("Deleting HostGrade: %s, HostId: %s\n", id, hg.HostId)

	// Delete the HostGrade
	_, err = kv.Delete(hostGradeKey, nil)
	if err != nil {
		return err
	}

	log.Println("HostGrade deleted successfully")

	// Remove the ID from the hostGradeIndex table
	indexKey := fmt.Sprintf(hostGradeIndex, hg.HostId)
	indexValue, _, err := kv.Get(indexKey, nil)
	if err != nil {
		return err
	}

	if indexValue != nil {
		log.Printf("Original indexValue: %s\n", indexValue.Value)

		updatedIndexValue := removeFromCSV(string(indexValue.Value), id)
		log.Printf("Updated indexValue: %s\n", updatedIndexValue)

		_, err := kv.Put(&api.KVPair{Key: indexKey, Value: []byte(updatedIndexValue)}, nil)
		if err != nil {
			return err
		}

		log.Println("HostGradeIndex updated successfully")
	} else {
		log.Println("HostGradeIndex not found")
	}

	return nil
}

func removeFromCSV(csvString, idToRemove string) string {
	log.Printf("csvString: %q\n", csvString)
	log.Printf("idToRemove: %q\n", idToRemove)
	var ids []string
	if err := json.Unmarshal([]byte(csvString), &ids); err != nil {
		log.Println("Error unmarshaling CSV string:", err)
	}

	var updatedIDs []string
	log.Println("ids", ids)
	log.Println("updatedIDs1", updatedIDs)
	for _, existingID := range ids {
		log.Println("Usli u for")
		existingID = strings.TrimSpace(existingID)
		idToRemove = strings.TrimSpace(idToRemove)
		if existingID != idToRemove {
			log.Println("existingID", existingID)
			updatedIDs = append(updatedIDs, existingID)
			log.Println("updatedIDs2", updatedIDs)
		}
	}
	log.Println("FinalUpdatedIDs", updatedIDs)
	jsonArrayString, err := json.Marshal(updatedIDs)
	if err != nil {
		log.Println("Error marshaling JSON array:", err)
		return ""
	}
	log.Println("FinalUpdatedIDs2", strings.Join(updatedIDs, ","))

	return string(jsonArrayString)
}

func (ur *UserRepo) UpdateHostGrade(hostId string, userId string, grade int) error {
	log.Println("Usli u UpdateHostGrade")
	log.Println("UserId: ", userId)
	log.Println("Grade:", grade)
	kv := ur.cli.KV()

	// Check the hostGradeIndex table by userId
	indexKey := fmt.Sprintf(hostGradeIndex, hostId)
	indexValue, _, err := kv.Get(indexKey, nil)
	if err != nil {
		return err
	}

	log.Println("hostGradeIndex sta je u njoj: ", indexValue)

	if indexValue == nil {
		return errors.New("no host grade found for the specified user")
	}

	var ids []string
	if err := json.Unmarshal(indexValue.Value, &ids); err != nil {
		return err
	}

	log.Println("Pre petlje")
	for _, id := range ids {
		log.Println("hostGradeIndex u for petlji: ", id)
		// Check if there is any data in the hostGrades table with the retrieved ID
		hostGradeKey := fmt.Sprintf(hostGrades, strings.TrimSpace(id))
		pair, _, err := kv.Get(hostGradeKey, nil)
		if err != nil {
			return err
		}

		log.Println("hostGrades tabela: ", pair)
		log.Println("Pre paira")

		if pair != nil {
			// Data found in hostGrades, update its grade
			var hg HostGrade
			if err := json.Unmarshal(pair.Value, &hg); err != nil {
				return err
			}

			log.Println("Id : ", userId)
			log.Println("UserId : ", hg.UserId)
			log.Println("HostGrade: ", hg)
			if strings.TrimSpace(hg.UserId) == strings.TrimSpace(userId) {
				hg.Grade = grade
				updatedData, err := json.Marshal(hg)
				if err != nil {
					return err
				}

				// Put the updated data back into the hostGrades table
				_, err = kv.Put(&api.KVPair{Key: hostGradeKey, Value: updatedData}, nil)
				if err != nil {
					return err
				}

				log.Printf("Grade updated successfully for HostGrade with ID: %s\n", id)
			}
		}
	}

	return nil
}

func (ur *UserRepo) GetAllReservatinsForUserByHostId(userId string, hostId string, ctx context.Context) (*http.Response, error) {
	ctx, span := ur.tracer.Start(ctx, "userRepo.GetAllReservatinsForUserByHostId")
	defer span.End()
	url := ur.reservation_service_string + "/api/reservations/by_user_for_host_id/" + userId + "/" + hostId

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

func (ur *UserRepo) GetAllHostGradesByHostId(hostId string, ctx context.Context) ([]HostGrade, error) {
	ctx, span := ur.tracer.Start(ctx, "userRepo.GetAllHostGradesByHostId")
	defer span.End()

	log.Println("Usli u GetAllHostGradesByHostId")
	kv := ur.cli.KV()

	indexKey := fmt.Sprintf(hostGradeIndex, strings.Trim(hostId, `"`))
	indexValue, _, err := kv.Get(indexKey, nil)
	if err != nil {
		return nil, err
	}

	log.Println("IndeKey", indexKey)
	log.Println("IndexValie", indexValue)

	if indexValue == nil {
		return nil, nil // No HostGrades found for the specified HostId
	}

	log.Println("Index value:", indexValue)

	var hostGradeIDs []string
	if err := json.Unmarshal(indexValue.Value, &hostGradeIDs); err != nil {
		return nil, err
	}
	log.Println("hostGradeIDs:", hostGradeIDs)
	hostGrades := make([]HostGrade, 0)
	log.Println("hostGrades:", hostGrades)

	for _, id := range hostGradeIDs {
		log.Println("hostGradeID:", id)
		hostGrade, err := ur.GetHostGradeByID(id)
		if err != nil {
			return nil, err
		}
		log.Println("hostGrade:", hostGrade)

		hostGrades = append(hostGrades, *hostGrade)
	}

	return hostGrades, nil
}

// GetHostGradeByID retrieves a host grade by its ID
func (ur *UserRepo) GetHostGradeByID(id string) (*HostGrade, error) {
	kv := ur.cli.KV()

	hostGradeKey := fmt.Sprintf(hostGrades, id)
	pair, _, err := kv.Get(hostGradeKey, nil)
	if err != nil {
		return nil, err
	}

	if pair == nil {
		return nil, errors.New("HostGrade not found")
	}

	log.Println("HostGrade:", pair)

	var hg HostGrade
	if err := json.Unmarshal(pair.Value, &hg); err != nil {
		return nil, err
	}

	return &hg, nil
}

func (uh *UserRepo) UpdateUserGrade(userId string, grade float64, ctx context.Context) (*http.Response, error) {
	ctx, span := uh.tracer.Start(ctx, "userRepo.UpdateUserGrade")
	defer span.End()

	url := uh.auth_service_string + "/api/users/updateGrade"

	requestData := AverageGrade{
		UserId:       userId,
		AverageGrade: grade,
	}

	reqBody, err := json.Marshal(&requestData)
	if err != nil {
		return nil, err
	}

	log.Println("Url:", url)
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
