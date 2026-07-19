package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/sashabaranov/go-openai"
)

func TestVectorQueryAgent(t *testing.T) {
    // Defines standard test cases
	tests := []struct {
		name     string
		question string
		expected string
	}{
		{
			name:     "TestPhil",
			question: "What courses is Phil Peterson teaching in Fall 2024?",
			expected: `Philip Peterson is teaching the following courses this Fall:
1. **CS 272L 01: Software Development Lab**
   - Course Code: 42343
   - Class Type: In-Person
   - Schedule: Wednesdays, 1:00 PM - 2:30 PM
   - Location: MH 122
   - Enrollment: 21 students
   - Email: phpeterson@usfca.edu
2. **SCCS 272 04: Software Development**
   - Course Code: 40647
   - Class Type: In-Person
   - Schedule: Tuesdays and Thursdays, 8:00 AM - 9:45 AM
   - Location: LS G12`,
		},
        // You can add the "TestPHIL", "TestBio", and "TestGuitar" cases here exactly as they were in V1
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			file, err := os.Open("courses.csv")
			if err != nil {
				log.Fatalf("error opening csv file: %v", err)
			}
			defer file.Close()

            // Initialize DB without resetting for test speed
			db, err := NewChromaDB(false)
			if err != nil {
				t.Fatalf("failed to create new chroma db: %v", err)
			}
			db.insertRecords(file, false)

			client := openai.NewClient(os.Getenv("OPENAI_PROJECT_KEY"))
			ctx := context.Background()
			tools := makeTools()

			dialogue := []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: "You are a helpful assistant...", // Keep concise for testing
				},
			}

			// Run the Agentic workflow
			answer, err := RunAgent(ctx, db, &dialogue, tools, test.question)
			if err != nil {
				t.Fatalf("Agent run failed: %v", err)
			}

            // LLM-as-a-Judge: Evaluate if the Agent's response semantically matches expectations
			bot := &chatBot{
				data: "Compare these two course listings and return ONLY 'yes' if the information is at least 50 percent similar. The only thing that matters are the courses, instructors and meeting times (ignoring formatting), or 'no' if they differ:\n\nFirst listing:\n" + answer + "\n\nSecond listing:\n" + test.expected,
				question: "Are these course listings equivalent in content? Answer ONLY 'yes' or 'no'.",
			}

			actual, err := bot.callOpenAI(client)
			if err != nil {
				t.Fatalf("API call failed: %v", err)
			}

			if actual != "yes" {
				t.Errorf("Results are not similar\nGot:\n%s\n\nExpected:\n%s", answer, test.expected)
			}
		})
	}
}