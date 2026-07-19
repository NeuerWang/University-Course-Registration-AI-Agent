package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	chroma "github.com/amikos-tech/chroma-go"
	"github.com/amikos-tech/chroma-go/openai"
	"github.com/amikos-tech/chroma-go/types"
	"github.com/joho/godotenv"
	gopenai "github.com/sashabaranov/go-openai"
)

type Database struct {
	client                *chroma.Client
	coursesCollection     *chroma.Collection
	instructorsCollection *chroma.Collection
	openaiEf              *openai.OpenAIEmbeddingFunction
	openaiClient          *gopenai.Client
}

func NewChromaDB(reset bool) (*Database, error) {
	if err := godotenv.Load("api.env"); err != nil {
		return nil, fmt.Errorf("err loadingenv file: %v", err)
	}

	key := os.Getenv("OPENAI_PROJECT_KEY")
	if key == "" {
		return nil, fmt.Errorf("api key not found")
	}

	ctx := context.Background()
	client, err := chroma.NewClient("http://0.0.0.0:8000")
	if err != nil {
		return nil, fmt.Errorf("err creating client: %s", err)
	}

	openaiEf, err := openai.NewOpenAIEmbeddingFunction(key)
	if err != nil {
		fmt.Printf("err initializing OpenAI embedding function: %v", err)
	}

	var courseCollection, instructorCollection *chroma.Collection

	if reset {
		_, err = client.DeleteCollection(ctx, "courses")
		if err != nil {
			fmt.Printf("Failed to delete courses collection: %v", err)
		}

		_, err = client.DeleteCollection(ctx, "instructors")
		if err != nil {
			fmt.Printf("Failed to delete instructors collection: %v", err)
		}

		_, err = client.CreateCollection(ctx, "courses", nil, true, openaiEf, types.L2)
		if err != nil {
			fmt.Printf("Failed to create courses collection: %v", err)
		}

		_, err = client.CreateCollection(ctx, "instructors", nil, true, openaiEf, types.L2)
		if err != nil {
			fmt.Printf("Failed to create instructors collection: %v", err)
		}
	}

	courseCollection, err = client.GetCollection(ctx, "courses", openaiEf)
	if err != nil {
		return nil, fmt.Errorf("failed to get courses collection: %v", err)
	}

	instructorCollection, err = client.GetCollection(ctx, "instructors", openaiEf)
	if err != nil {
		return nil, fmt.Errorf("failed to get instructors collection: %v", err)
	}

	return &Database{
		client:                client,
		coursesCollection:     courseCollection,
		instructorsCollection: instructorCollection,
		openaiEf:              openaiEf,
		openaiClient:          gopenai.NewClient(key),
	}, nil
}

func (db *Database) insertRecords(r io.Reader, reset bool) error {
	if !reset {
		return nil
	}

	records, err := ReadCSV(r)
	if err != nil {
		return err
	}

	if len(records) < 1 {
		return fmt.Errorf("no records to insert")
	}

	fmt.Printf("processing %d records from CSV\n", len(records)-1)

	instructors := make(map[string]bool)
	instructorsNames := []string{}
	var courseDocuments []courseRecord

	for _, row := range records[1:] {
		firstName := row[17]
		lastName := row[18]
		fullName := firstName + " " + lastName
		if !instructors[fullName] {
			instructors[fullName] = true
			instructorsNames = append(instructorsNames, fullName)
		}
		courseDocuments = append(courseDocuments, courseRecord{
			document:   strings.Join(row, " "),
			instructor: fullName,
		})
	}

	fmt.Printf("Found %d instructors and %d courses\n", len(instructors), len(courseDocuments))

	if err := db.insertCourses(courseDocuments); err != nil {
		return fmt.Errorf("failed to insert courses: %v", err)
	}

	if err := db.insertInstructors(instructorsNames); err != nil {
		return fmt.Errorf("failed to insert instructors: %v", err)
	}

	return nil
}
