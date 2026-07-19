# University Course Registration AI Agent

> An intelligent, terminal-based conversational agent built in Go that helps students navigate complex university course schedules using an Agentic Workflow and Semantic Search.

## Project Overview

This project is a CLI-based AI assistant designed to streamline the course discovery process. Moving beyond traditional, static RAG (Retrieval-Augmented Generation) pipelines, this system implements a modern **Agentic Architecture**. It utilizes OpenAI's Tool Calling capabilities to dynamically orchestrate semantic database queries, handle fuzzy data matching, and even execute local system commands (e.g., drafting emails) based on natural language multi-turn conversations.

## Core Engineering Highlights

*   **Agentic Workflow via Tool Calling:** Replaced hard-coded prompts with strict JSON Schema tool definitions (`extract_name`, `query_courses`, `open_email`). The LLM acts as a reasoning engine, deciding when and how to invoke local Go functions.
*   **Stateful Multi-Turn Dialogue:** Implemented a robust context-management system within the CLI loop, allowing the agent to remember previous turns and handle complex follow-up queries seamlessly.
*   **Semantic Search & Fuzzy Matching:** Integrated **ChromaDB** for vector storage. To handle real-world "dirty data", the system implements a custom fuzzy-matching scoring algorithm to bridge the gap between user typos and canonical instructor names in the database.
*   **"LLM-as-a-Judge" Unit Testing:** Built an automated testing suite (`vectorQuery_test.go`) that uses an LLM evaluator to assert the semantic accuracy of the Agent's responses against expected outputs, ensuring production-level reliability.

## Tech Stack

*   **Language:** Go (Golang 1.23)
*   **AI/LLM:** OpenAI API (`go-openai`), GPT-4o-mini, Function Calling
*   **Vector Database:** ChromaDB (Dockerized)
*   **Data Processing:** Custom CSV parser & embedding generator

---

## Getting Started (Local Development)

### 1. Prerequisites
*   [Go](https://golang.org/doc/install) (1.23 or later)
*   [Docker Desktop](https://www.docker.com/products/docker-desktop)
*   An OpenAI API Key

### 2. Environment Setup
Create an `api.env` file in the root directory and add your OpenAI API key:
```env
OPENAI_PROJECT_KEY=your_openai_api_key_here
```

### 3. Start the Vector Database
Launch the persistent ChromaDB instance via Docker Compose:
```bash
docker-compose up -d
```

### 4. Data Ingestion & Execution
To parse the `courses.csv`, generate embeddings, and start the interactive chat interface, run:
```bash
# Use the -reset flag for the first run to build the ChromaDB collections
go run . -reset

# For subsequent runs (faster, skips ingestion):
go run .
```

---

## Testing

The project includes an automated test suite verifying the RAG retrieval accuracy and the Agent's tool-calling logic. Run the tests using:
```bash
go test -v
```

## Repository Structure

*   `main.go` - Entry point and stateful Agent dialogue loop.
*   `db.go` - ChromaDB client initialization and collection management.
*   `query.go` - Tool definitions (JSON Schema) and OpenAI tool execution logic.
*   `insertCollections.go` - Batch processing and embedding logic for CSV ingestion.
*   `vectorQuery_test.go` - LLM-as-a-Judge unit tests.