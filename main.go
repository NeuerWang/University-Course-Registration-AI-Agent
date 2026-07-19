package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/sashabaranov/go-openai"
)

// RunAgent executes a single turn of the agentic loop.
// Abstracting this logic out of main() allows for robust unit testing.
func RunAgent(ctx context.Context, db *Database, dialogue *[]openai.ChatCompletionMessage, tools []openai.Tool, question string) (string, error) {
	// Append the new user question to the conversation history
	*dialogue = append(*dialogue, openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleUser,
		Content: question,
	})

	var fullResponse string
	continueTool := true

	for continueTool {
		resp, err := db.openaiClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model:    openai.GPT4oMini,
			Messages: *dialogue,
			Tools:    tools,
		})
		if err != nil {
			return fullResponse, fmt.Errorf("error getting completion: %v", err)
		}

		message := resp.Choices[0].Message
		*dialogue = append(*dialogue, message)

		// If the model decides no more tools are needed, break the loop
		if len(message.ToolCalls) == 0 {
			if fullResponse != "" {
				fullResponse += "\n" + message.Content
			} else {
				fullResponse = message.Content
			}
			continueTool = false
			break
		}

		// Handle all tool calls requested by the model
		for _, toolCall := range message.ToolCalls {
			result, err := db.handleToolCall(ctx, toolCall)
			if err != nil {
				fmt.Printf("Error handling tool call: %v\n", err)
				continue
			}

			// Append the tool's execution result back to the dialogue context
			toolMessage := openai.ChatCompletionMessage{
				Role:       openai.ChatMessageRoleTool,
				Content:    result,
				Name:       toolCall.Function.Name,
				ToolCallID: toolCall.ID,
			}
			*dialogue = append(*dialogue, toolMessage)

			// Do not append intermediate extraction logs to the final response
			if toolCall.Function.Name != "extract_name" && result != "No matching courses found" {
				if fullResponse != "" {
					fullResponse += "\n" + result
				} else {
					fullResponse = result
				}
			}
		}
	}
	return fullResponse, nil
}

func main() {
	reset := flag.Bool("reset", false, "set true to reset database")
	flag.Parse()

	file, err := os.Open("courses.csv")
	if err != nil {
		log.Fatalf("error opening csv file: %v", err)
	}
	defer file.Close()

	db, err := NewChromaDB(*reset)
	if err != nil {
		log.Fatalf("failed to create new chroma db: %v", err)
	}
	if err := db.insertRecords(file, *reset); err != nil {
		log.Fatalf("failed to insert records: %v", err)
	}

	tools := makeTools()
	scanner := bufio.NewScanner(os.Stdin)
	ctx := context.Background()

	// Initialize the system prompt and conversation history
	dialogue := []openai.ChatCompletionMessage{
		{
			Role: openai.ChatMessageRoleSystem,
			Content: `You are a helpful assistant with access to a university course database and can help with email tasks. 
You can answer questions about courses and instructors, and can open email services in the web browser. 
When asked about email or sending messages, offer to open a web email service (Gmail, Outlook, or Yahoo). 
When asked about multiple instructors, make separate queries for each instructor and combine the results. 
If you can't find an exact match for an instructor name, try to find the closest match. 
If you can't find any courses for an instructor, clearly indicate that and offer to help find alternatives. 
Use markdown formatting for headers and course listings to make the information clear and readable. 
Maintain context from previous questions to handle follow-up queries naturally.`,
		},
	}

	fmt.Printf("\nEnter your question> ")
	for scanner.Scan() {
		question := scanner.Text()
		
		response, err := RunAgent(ctx, db, &dialogue, tools, question)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Println(response)
		}
		
		fmt.Print("\nEnter your question> ")
	}
}