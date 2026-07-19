package main

import (
	"encoding/csv"
	"fmt"
	"io"
)

func ReadCSV(r io.Reader) ([][]string, error) {
	reader := csv.NewReader(r)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read the csv file: %v", err)
	}
	return records, nil
}
