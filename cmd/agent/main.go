package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/geminitool"
	"google.golang.org/genai"
)

func main() {
	ctx := context.Background()

	model, err := gemini.NewModel(ctx, "gemini-2.5-flash", &genai.ClientConfig{
		APIKey: os.Getenv("GOOGLE_API_KEY"),
	})
	if err != nil {
		log.Fatalf("Failed to create model: %v", err)
	}

	timeAgent, err := llmagent.New(llmagent.Config{
		Name:        "hello_time_agent",
		Model:       model,
		Description: "Tells the current time in a specified city.",
		Instruction: "You are a helpful assistant that tells the current time in a city.",
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Create session service
	sessionService := session.InMemoryService()

	// Create runner
	r, err := runner.New(runner.Config{
		AppName:        "time_app",
		Agent:          timeAgent,
		SessionService: sessionService,
	})
	if err != nil {
		log.Fatalf("Failed to create runner: %v", err)
	}

	// Create a session first
	userID := "user123"
	sess, err := sessionService.Create(ctx, &session.CreateRequest{
		AppName: "time_app",
		UserID:  userID,
	})
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	// Create user message with "shenzhen"
	userMessage := &genai.Content{
		Role: "user",
		Parts: []*genai.Part{
			{Text: "What is the current time in shenzhen?"},
		},
	}

	// Run agent with streaming
	runConfig := agent.RunConfig{
		StreamingMode: agent.StreamingModeSSE,
	}

	fmt.Println("Querying time for shenzhen...")
	for event, err := range r.Run(ctx, userID, sess.Session.ID(), userMessage, runConfig) {
		if err != nil {
			log.Fatalf("Error during run: %v", err)
		}
		// Process events
		if event.Content != nil {
			for _, part := range event.Content.Parts {
				fmt.Printf("%s", part.Text)
			}
		}
	}
	fmt.Println()
}
