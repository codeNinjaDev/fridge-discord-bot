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
	dg.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		// Pass the interaction to the global handler
		HandleInteraction(s, i)
	})

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
				_, embed, info, err := ProcessSingleFood(ctx, model, attachment)
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
					s.ChannelMessageSendEmbed(m.ChannelID, embed)
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
				_, embed, _, err := ProcessSingleFood(ctx, model, attachment)
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, "Failed to process image 😢")
				} else {
					s.ChannelMessageSendEmbed(m.ChannelID, embed)
				}
			}
		} else {
			fmt.Println("No attachments in the message.")
		}
	}).setHelp("Queries food").add()

	newCommand("!get", 0, false, func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
		foods, err := food_db.GetAllFoods(m.Author.ID)
		var embeds []*discordgo.MessageEmbed
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to get foods 😢")
		}
		for _, food := range foods {
			// markdown, err := PrettyMarkdownResponse(food)
			// if err != nil {
			// 	log.Error("Could not get food: %v", err)
			// }
			// s.ChannelMessageSend(m.ChannelID, markdown)

			embed, err := CreateFoodEmbed(food)
			if err != nil {
				log.Error("Could not get food: %v", err)
			} else {
				embeds = append(embeds, embed)
			}
		}

		slider, err := NewSlider(s, m.ChannelID, embeds)
		if err != nil {
			s.ChannelMessageSend(m.ChannelID, "Failed to get foods 😢")
		} else {
			slider.Send()
		}
	}).setHelp("Queries food").add()

	newCommand("!askall", 0, false, func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
		if len(m.Attachments) > 0 {
			for _, attachment := range m.Attachments {

				err := s.ChannelTyping(m.ChannelID)
				if err != nil {
					log.Error("Error sending typing indicator: %v", err)
				}
				_, embeds, _, err := ProcessMultipleFoods(ctx, model, attachment)
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, "Failed to process image 😢")
				} else {
					slider, err := NewSlider(s, m.ChannelID, embeds)
					if err != nil {
						s.ChannelMessageSend(m.ChannelID, "Failed to make slider")
					} else {
						slider.Send()
					}
				}
			}
		} else {
			fmt.Println("No attachments in the message.")
		}
	}).setHelp("Looks for all food in the image").add()

	newCommand("!scanall", 0, false, func(s *discordgo.Session, m *discordgo.MessageCreate, args []string) {
		if len(m.Attachments) > 0 {
			for _, attachment := range m.Attachments {

				err := s.ChannelTyping(m.ChannelID)
				if err != nil {
					log.Error("Error sending typing indicator: %v", err)
				}
				_, embeds, foods, err := ProcessMultipleFoods(ctx, model, attachment)
				if err != nil {
					s.ChannelMessageSend(m.ChannelID, "Failed to process image 😢")
				} else {
					for _, food := range foods {
						food.UserId = m.Author.ID
						err := food_db.CreateFood(&food)
						if err != nil {
							log.Error("Error creating food entry in db: %v", err)
						}
					}
					slider, err := NewSlider(s, m.ChannelID, embeds)
					if err != nil {
						s.ChannelMessageSend(m.ChannelID, "Failed to make slider")
					} else {
						slider.Send()
					}
				}
			}
		} else {
			fmt.Println("No attachments in the message.")
		}
	}).setHelp("Looks for all food in the image").add()

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

func ProcessSingleFood(ctx context.Context, model *genai.GenerativeModel, attachment *discordgo.MessageAttachment) (string, *discordgo.MessageEmbed, *FoodInfo, error) {
	if strings.HasPrefix(attachment.ContentType, "image/") {
		fmt.Printf("Attachment is an image: <%s>\n", attachment.URL)
		extension := filepath.Ext(attachment.Filename)

		result, err := generateFoodString(ctx, model, attachment.URL, extension)
		if result != "" {
			info, err := UnmarshalFood(result)
			if err != nil {
				log.Error("Error marshalling: %v", err)
				log.Error("Raw result: %s", result)
				return "", nil, nil, err
			}
			info.ImageUrl = attachment.URL
			markdown, err := PrettyMarkdownResponse(*info)
			if err != nil {
				log.Error("Error parsing markdown: %v", err)
				log.Error("Raw result: %s", result)
				return "", nil, info, err
			}
			embed, err := CreateFoodEmbed(*info)
			return markdown, embed, info, err

		} else {
			log.Error("Failed to process image content: %v", err)
		}
	}
	return "", nil, nil, fmt.Errorf("attachment is not an image: %v", attachment.URL)
}

func ProcessMultipleFoods(ctx context.Context, model *genai.GenerativeModel, attachment *discordgo.MessageAttachment) ([]string, []*discordgo.MessageEmbed, []FoodInfo, error) {
	var markdowns []string
	var foods []FoodInfo
	var embeds []*discordgo.MessageEmbed
	if strings.HasPrefix(attachment.ContentType, "image/") {
		fmt.Printf("Attachment is an image: <%s>\n", attachment.URL)
		extension := filepath.Ext(attachment.Filename)

		result, err := generateAllFoodStrings(ctx, model, attachment.URL, extension)
		if result != "" {
			var err error
			foods, err = UnmarshalFoods(result)
			if err != nil {
				log.Error("Error marshalling: %v", err)
				log.Error("Raw result: %s", result)
				return nil, nil, nil, err
			}

			for _, food := range foods {
				food.ImageUrl = attachment.URL
				markdown, err := PrettyMarkdownResponse(food)
				if err != nil {
					log.Error("Could not get food: %v", err)
				} else {
					markdowns = append(markdowns, markdown)

				}
				embed, err := CreateFoodEmbed(food)
				if err != nil {
					log.Error("Could not create embed for food: %v", err)
				} else {
					embeds = append(embeds, embed)
				}

			}

			return markdowns, embeds, foods, err

		} else {
			log.Error("Failed to process image content: %v", err)
		}
	}
	return markdowns, embeds, foods, fmt.Errorf("attachment is not an image: %v", attachment.URL)
}
