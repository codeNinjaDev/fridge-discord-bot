package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/google/generative-ai-go/genai"
	"github.com/joho/godotenv"
	"google.golang.org/api/option"
)

var (
	log = newLog()
)
var ctx = context.Background()

// Access your API key as an environment variable
var client *genai.Client
var gemini_err error

func main() {
	defer client.Close()
	token := os.Getenv("BOT_TOKEN")
	if token == "" {
		fmt.Println("Please set the DISCORD_BOT_TOKEN environment variable.")
		return
	}

	// Create a new Discord session
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session:", err)
		return
	}

	// Register a message handler
	dg.AddHandler(messageHandler)

	// Open a websocket connection to Discord
	err = dg.Open()
	if err != nil {
		fmt.Println("Error opening connection:", err)
		return
	}

	fmt.Println("Bot is now running. Press CTRL+C to exit.")
	// Wait for a termination signal
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	// Cleanly close the Discord session
	dg.Close()
}

// messageHandler processes incoming messages and routes commands
func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from the bot itself
	if m.Author.ID == s.State.User.ID {
		return
	}

	// Process the command using the helper file
	parseCommand(s, m, &discordgo.Guild{}, m.Content)
}

func init() {

	godotenv.Load()
	api_key := os.Getenv("GEMINI_API")
	client, gemini_err = genai.NewClient(ctx, option.WithAPIKey(api_key))
	if gemini_err != nil {
		panic(gemini_err)
	}

	var model = client.GenerativeModel("gemini-2.0-flash-exp")

	db_host := os.Getenv("DB_FILE")
	food_db, err := NewFoodService(db_host)
	if err != nil {
		panic(err)
	}

	// Add a simple ping command
	newCommand("!ping", 0, false, func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
		s.ChannelMessageSend(m.ChannelID, "Pong!")
	}).setHelp("Responds with Pong!").add()

	// Add an echo command
	newCommand("!echo", 0, false, func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
		if len(args) > 1 {
			message := args[1:]
			s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Echo: %s", message))
		} else {
			s.ChannelMessageSend(m.ChannelID, "Usage: !echo <message>")
		}
	}).setHelp("Repeats the message you send.").add()

	// Add an scan command
	newCommand("!scan", 0, false, func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
		if len(m.Attachments) > 0 {
			for _, attachment := range m.Attachments {

				err := s.ChannelTyping(m.ChannelID)
				if err != nil {
					log.Error("Error sending typing indicator: %v", err)
				}
				markdown, info, err := ProcessAttachment(ctx, s, m, model, attachment)
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, "Failed to process image 😢")
				} else {
					info.UserId = m.Author.ID
					err := food_db.CreateFood(info)
					if err != nil {
						s.ChannelMessageSend(m.ChannelID, "Failed to create entry 😢")
						log.Error("Error creating food entry in db: %v", err)
						return
					}
					s.ChannelMessageSend(m.ChannelID, markdown)
				}
			}
		} else {
			fmt.Println("No attachments in the message.")
		}
	}).setHelp("Scans food into database").add()

	newCommand("!ask", 0, false, func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
		if len(m.Attachments) > 0 {
			for _, attachment := range m.Attachments {

				err := s.ChannelTyping(m.ChannelID)
				if err != nil {
					log.Error("Error sending typing indicator: %v", err)
				}
				markdown, _, err := ProcessAttachment(ctx, s, m, model, attachment)
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, "Failed to process image 😢")
				} else {
					s.ChannelMessageSend(m.ChannelID, markdown)
				}
			}
		} else {
			fmt.Println("No attachments in the message.")
		}
	}).setHelp("Queries food").add()

	newCommand("!get", 0, false, func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
		foods, err := food_db.GetAllFoods(m.Author.ID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to get foods 😢")
		}
		for _, food := range foods {
			markdown, err := PrettyMarkdownResponse(food)
			if err != nil {
				log.Error("Could not get food: %v", err)
			}
			s.ChannelMessageSend(m.ChannelID, markdown)

		}
	}).setHelp("Queries food").add()

	newCommand("!clearall", 0, false, func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
		foods, err := food_db.GetAllFoods(m.Author.ID)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to get foods 😢")
		}

		var markdown bytes.Buffer
		markdown.WriteString("Deleting foods: \n")
		for _, food := range foods {
			err := food_db.DeleteFood(food.ID)
			if err == nil {
				markdown.WriteString(fmt.Sprintf("\t- %s\n", food.FoodItem))
			}
		}
		s.ChannelMessageSend(m.ChannelID, markdown.String())
	}).setHelp("Queries food").add()
}

func ProcessAttachment(ctx context.Context, s *discordgo.Session, m *discordgo.MessageCreate, model *genai.GenerativeModel, attachment *discordgo.MessageAttachment) (string, *FoodInfo, error) {
	if strings.HasPrefix(attachment.ContentType, "image/") {
		fmt.Printf("Attachment is an image: <%s>\n", attachment.URL)
		extension := filepath.Ext(attachment.Filename)

		result, err := generateContent(ctx, model, attachment.URL, extension)
		if result != "" {
			info, err := MarshalFood(result)
			if err != nil {
				log.Error("Error marshalling: %v", err)
				log.Error("Raw result: %s", result)
				return "", nil, err
			}
			markdown, err := PrettyMarkdownResponse(*info)
			if err != nil {
				log.Error("Error parsing markdown: %v", err)
				log.Error("Raw result: %s", result)
			}
			return markdown, info, err

		} else {
			log.Error("Failed to process image content: %v", err)
		}
	}
	return "", nil, fmt.Errorf("Attachment is not an image: %v", attachment.URL)
}
