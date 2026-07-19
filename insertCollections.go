package main

import (
	"context"
	"fmt"

	"github.com/amikos-tech/chroma-go/types"
)

type courseRecord struct {
	document   string
	instructor string
}

func (db *Database) insertCourses(courses []courseRecord) error {
	ctx := context.Background()
	fmt.Printf("attempting to insert %d courses\n", len(courses))
	if len(courses) == 0 {
		fmt.Println("nothing to insert")
		return nil
	}

	const batchSize = 1000
	for i := 0; i < len(courses); i += batchSize {
		end := i + batchSize
		if end > len(courses) {
			end = len(courses)
		}

		rs, err := types.NewRecordSet(
			types.WithEmbeddingFunction(db.coursesCollection.EmbeddingFunction),
			types.WithIDGenerator(types.NewUUIDGenerator()),
		)
		if err != nil {
			return fmt.Errorf("err creating course record set: %v", err)
		}

		for _, course := range courses[i:end] {
			rs.WithRecord(
				types.WithDocument(course.document),
				types.WithMetadata("instructor", course.instructor),
			)
		}

		if _, err := rs.BuildAndValidate(ctx); err != nil {
			return fmt.Errorf("err building and validating courses: %v", err)
		}
		if _, err := db.coursesCollection.AddRecords(ctx, rs); err != nil {
			return fmt.Errorf("err adding courses: %v", err)
		}
	}
	return nil
}

func (db *Database) insertInstructors(instructorNames []string) error {
	ctx := context.Background()
	if len(instructorNames) == 0 {
		return nil
	}

	irs, err := types.NewRecordSet(
		types.WithEmbeddingFunction(db.instructorsCollection.EmbeddingFunction),
		types.WithIDGenerator(types.NewUUIDGenerator()),
	)
	if err != nil {
		return fmt.Errorf("err creating instructor record set: %v", err)
	}

	for _, instructor := range instructorNames {
		irs.WithRecord(types.WithDocument(instructor))
		fmt.Printf("added instructor: %s\n", instructor)
	}

	_, err = irs.BuildAndValidate(ctx)
	if err != nil {
		return fmt.Errorf("err building and validating instructors: %v", err)
	}
	fmt.Println("finished B and V")

	fmt.Println("It is time to add records")
	_, err = db.instructorsCollection.AddRecords(ctx, irs)
	if err != nil {
		return fmt.Errorf("err adding instructors: %v", err)
	}
	fmt.Println("Finished Adding")

	return nil
}
