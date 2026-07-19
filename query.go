package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

type chatBot struct {
	question string
	data     string
}

type NameExtractRequest struct {
	Name string `json:"name"`
}

type CourseQueryRequest struct {
	Instructor string `json:"instructor,omitempty"`
	Question   string `json:"question"`
}

type EmailRequest struct {
	Service string `json:"service"`
}

func (bot *chatBot) callOpenAI(client *openai.Client) (string, error) {
	req := openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleSystem,
				Content: bot.data,
			},
			{
				Role:    openai.ChatMessageRoleUser,
				Content: bot.question,
			},
		},
	}

	resp, err := client.CreateChatCompletion(context.TODO(), req)
	if err != nil {
		fmt.Println("failed to create chat completion: ", err)
	}
	return resp.Choices[0].Message.Content, nil
}

func makeTools() []openai.Tool {
	nameParams := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"name": {
				Type:        jsonschema.String,
				Description: "The extracted name from the question, or 'none' if no name is found",
			},
		},
		Required: []string{"name"},
	}

	extractNameTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "extract_name",
			Description: "Extract instructor name from the question if present",
			Parameters:  nameParams,
		},
	}

	courseParams := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"instructor": {
				Type:        jsonschema.String,
				Description: "The canonical instructor name to query for",
			},
			"question": {
				Type:        jsonschema.String,
				Description: "The original question to use for semantic search",
			},
		},
		Required: []string{"question"},
	}

	queryCoursesTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "query_courses",
			Description: "Query the course database using instructor name and/or question",
			Parameters:  courseParams,
		},
	}

	emailParams := jsonschema.Definition{
		Type: jsonschema.Object,
		Properties: map[string]jsonschema.Definition{
			"service": {
				Type:        jsonschema.String,
				Enum:        []string{"gmail", "outlook", "yahoo"},
				Description: "The email service to open (gmail, outlook, or yahoo)",
			},
		},
		Required: []string{"service"},
	}

	emailTool := openai.Tool{
		Type: openai.ToolTypeFunction,
		Function: &openai.FunctionDefinition{
			Name:        "open_email",
			Description: "Opens the specified email service in the default web browser",
			Parameters:  emailParams,
		},
	}

	return []openai.Tool{
		extractNameTool,
		queryCoursesTool,
		emailTool,
	}
}

func (db *Database) handleToolCall(ctx context.Context, tool openai.ToolCall) (string, error) {
	switch tool.Function.Name {
	case "extract_name":
		var req NameExtractRequest
		if err := json.Unmarshal([]byte(tool.Function.Arguments), &req); err != nil {
			return "", fmt.Errorf("failed to unmarshal name request: %v", err)
		}
		return db.extractName(ctx, req.Name)

	case "query_courses":
		var req CourseQueryRequest
		if err := json.Unmarshal([]byte(tool.Function.Arguments), &req); err != nil {
			return "", fmt.Errorf("failed to unmarshal query request: %v", err)
		}
		return db.queryCourses(ctx, req.Instructor, req.Question)

	case "open_email":
		var req EmailRequest
		if err := json.Unmarshal([]byte(tool.Function.Arguments), &req); err != nil {
			return "", fmt.Errorf("failed to unmarshal email request: %v", err)
		}
		return db.openEmail(req.Service)

	default:
		return "", fmt.Errorf("unknown tool: %s", tool.Function.Name)
	}
}

func (db *Database) extractName(ctx context.Context, text string) (string, error) {
	bot := &chatBot{
		question: text,
		data:     "Extract only the first name and last name from the following text. If no name is present, respond with 'none' as the answer. If there's a shortened name like 'Phil', expand it to the full name 'Philip'.",
	}

	name, err := bot.callOpenAI(db.openaiClient)
	if err != nil {
		return "", err
	}

	if name != "none" {
		//Get canonical name from instructor collection with fuzzy matching
		result, err := db.instructorsCollection.Query(ctx, []string{name}, 5, nil, nil, nil)
		if err != nil {
			return "", err
		}

		if len(result.Documents) > 0 {
			//Look for best match among returned names
			bestMatch := ""
			bestScore := 0
			searchName := strings.ToLower(name)

			for _, docs := range result.Documents {
				for _, doc := range docs {
					//A string matching score
					score := 0
					docLower := strings.ToLower(doc)

					//Exact match gets highest score
					if docLower == searchName {
						bestMatch = doc
						break
					}

					//Check if all parts of the search name are in the document name
					searchParts := strings.Fields(searchName)
					for _, part := range searchParts {
						if strings.Contains(docLower, part) {
							score++
						}
					}

					if score > bestScore {
						bestScore = score
						bestMatch = doc
					}
				}
				if bestMatch != "" {
					break
				}
			}

			if bestMatch != "" {
				name = bestMatch
			}
		}
	}

	return name, nil
}

func (db *Database) queryCourses(ctx context.Context, instructor, question string) (string, error) {
	if instructor == "" || instructor == "none" {
		return db.queryWithoutInstructor(ctx, question)
	}

	metadata := map[string]interface{}{"instructor": instructor}
	results, err := db.coursesCollection.Query(ctx, []string{question}, 10, metadata, nil, nil)
	if err != nil {
		return "", err
	}

	if len(results.Documents) == 0 {
		return fmt.Sprintf("### %s's Courses:\nNo courses found for %s", instructor, instructor), nil
	}

	response := fmt.Sprintf("### %s's Courses:\n", instructor)
	for i, doc := range results.Documents[0] {
		response += fmt.Sprintf("%d. %s\n", i+1, formatCourseInfo(doc))
	}

	return response, nil
}

func (db *Database) queryWithoutInstructor(ctx context.Context, question string) (string, error) {
	results, err := db.coursesCollection.Query(ctx, []string{question}, 10, nil, nil, nil)
	if err != nil {
		return "", err
	}

	if len(results.Documents) == 0 {
		return "No matching courses found", nil
	}

	var response string
	for i, doc := range results.Documents[0] {
		response += fmt.Sprintf("%d. %s\n", i+1, formatCourseInfo(doc))
	}

	return response, nil
}

func formatCourseInfo(rawInfo string) string {
	return rawInfo
}

func (db *Database) openEmail(service string) (string, error) {
	var url string
	switch service {
	case "gmail":
		url = "https://mail.google.com"
	case "outlook":
		url = "https://outlook.live.com"
	case "yahoo":
		url = "https://mail.yahoo.com"
	default:
		return "", fmt.Errorf("unsupported email service: %s", service)
	}

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin": //macOS
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default: //Linux and others
		cmd = exec.Command("xdg-open", url)
	}

	err := cmd.Start()
	if err != nil {
		return "", fmt.Errorf("failed to open browser: %v", err)
	}

	return fmt.Sprintf("Opening %s in your default browser...", service), nil
}
