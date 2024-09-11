package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	discordgo "github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type Server struct {
	dg *discordgo.Session
}

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	botToken := os.Getenv("BOT_TOKEN")
	openaiApiKey := os.Getenv("OPENAI_API_KEY")
	if openaiApiKey == "" {
		log.Fatal("OPENAI_API_KEY is required")
	}

	// Create a new Discord session using the environment variables
	dg, err := discordgo.New("Bot " + botToken)
	if err != nil {
		log.Fatal(err)
	}
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == s.State.User.ID {
			return
		}
		if m.Content == "ping" {
			s.ChannelMessageSend(m.ChannelID, "Monkey!")
			return
		}
		generateResponse := func(prompt string) ([]string, error) {
			client := openai.NewClient(option.WithAPIKey(openaiApiKey))
			chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
				Messages: openai.F([]openai.ChatCompletionMessageParamUnion{openai.UserMessage("You: " + prompt)}),
				Model:    openai.F(openai.ChatModelGPT4oMini),
			})
			if err != nil {
				return nil, fmt.Errorf("failed to generate response: %w", err)
			}
			var responses []string
			for _, message := range chatCompletion.Choices {
				if message.Message.Role == "assistant" {
					responses = append(responses, message.Message.Content)
				}
			}
			return responses, nil
		}
		response, err := generateResponse(m.ContentWithMentionsReplaced())
		if err != nil {
			log.Println(err)
			return
		}
		for _, r := range response {
			msg := fmt.Sprintf("Monkey: %s", r)
			if len(msg) > 2000 {
				messages := strings.Split(msg, "\n")
				for _, message := range messages {
					s.ChannelMessageSend(m.ChannelID, message)
				}
			}

			s.ChannelMessageSend(m.ChannelID, msg)
		}
	})
	// Start the Discord session
	err = dg.Open()
	if err != nil {
		log.Fatal(err)
	}

	defer dg.Close()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc
}
