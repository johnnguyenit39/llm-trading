package mongodb

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func NewConnection() (*mongo.Database, error) {

	// Use the SetServerAPIOptions() method to set the version of the Stable API on the client
	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	userName := os.Getenv("MONGO_DB_USER_NAME")
	dbPassword := os.Getenv("MONGO_DB_PASSWORD")
	mongoDbUrl := os.Getenv("MONGO_DB_URL")
	url := fmt.Sprintf("mongodb+srv://%s:%s@%s", userName, dbPassword, mongoDbUrl)
	opts := options.Client().ApplyURI(url).
		SetServerAPIOptions(serverAPI)

	// Create a new client and connect to the server
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		log.Fatal(err)
	}
	dbName := os.Getenv("MONGO_DB_NAME")
	database := client.Database(dbName)
	// Send a ping to confirm a successful connection to the "novel-db" database
	if err := database.RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Pinged your deployment. You successfully connected to MongoDB!")
	return database, err
}
