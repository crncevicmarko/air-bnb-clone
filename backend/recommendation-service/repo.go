package main

import (
	"context"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	log "github.com/sirupsen/logrus"
	"os"
)

type RecommendRepo struct {
	// Thread-safe instance which maintains a database connection pool
	driver neo4j.DriverWithContext
	logger *log.Logger
}

func New(logger *log.Logger) (*RecommendRepo, error) {
	// Local instance
	uri := os.Getenv("NEO4J_DB")
	user := os.Getenv("NEO4J_USERNAME")
	pass := os.Getenv("NEO4J_PASS")
	auth := neo4j.BasicAuth(user, pass, "")

	driver, err := neo4j.NewDriverWithContext(uri, auth)
	if err != nil {
		logger.Panic(err)
		return nil, err
	}

	// Return repository with logger and DB session
	return &RecommendRepo{
		driver: driver,
		logger: logger,
	}, nil
}

func (rr *RecommendRepo) CheckConnection() {
	ctx := context.Background()
	err := rr.driver.VerifyConnectivity(ctx)
	if err != nil {
		rr.logger.Panic(err)
		return
	}
	// Print Neo4J server address
	rr.logger.Printf(`Neo4J server address: %s`, rr.driver.Target().Host)
}

func (rr *RecommendRepo) CloseDriverConnection(ctx context.Context) {
	rr.driver.Close(ctx)
}

func (rr *RecommendRepo) WriteUser(recommend *Recommend) error {

	ctx := context.Background()
	session := rr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	_, err := session.ExecuteWrite(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				"MERGE (u:User {username: $username})"+
					"MERGE (a:Accommodation {id: $id})"+
					"MERGE (u)-[:RESERVED]->(a)",
				map[string]any{"username": recommend.Username, "id": recommend.ID})
			if err != nil {
				return nil, err
			}

			if result.Next(ctx) {
				return result.Record().Values[0], nil
			}

			return nil, result.Err()
		})
	if err != nil {
		rr.logger.Println("Error inserting Person:", err)
		return err
	}
	//rr.logger.Println(savedUser.(string))
	return nil
}

func (rr *RecommendRepo) GetAllRecommendations(username string) ([]string, error) {
	ctx := context.Background()
	session := rr.driver.NewSession(ctx, neo4j.SessionConfig{DatabaseName: "neo4j"})
	defer session.Close(ctx)

	// ExecuteRead for read transactions (Read and queries)
	AccommodationResults, err := session.ExecuteRead(ctx,
		func(transaction neo4j.ManagedTransaction) (any, error) {
			result, err := transaction.Run(ctx,
				`MATCH (u:User {username: $username})-[:RESERVED]->(a:Accommodation)<-[:RESERVED]-(o:User),
						(o:User)-[:RESERVED]->(rec:Accommodation)
						RETURN distinct rec.id as id
						LIMIT $limit`,
				map[string]any{"username": username, "limit": 10})
			if err != nil {
				return nil, err
			}

			var accoIds []string
			for result.Next(ctx) {
				record := result.Record()
				id, ok := record.Get("id")
				if !ok || id == nil {
					id = ""
				}
				accoIds = append(accoIds, id.(string))
			}
			return accoIds, nil
		})
	if err != nil {
		rr.logger.Println("Error querying search:", err)
		return nil, err
	}
	rr.logger.Println(AccommodationResults)
	return AccommodationResults.([]string), nil
}
