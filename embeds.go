package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var sliders = sync.Map{} // A thread-safe map to store multiple sliders

type Slider struct {
	Embeds     []*discordgo.MessageEmbed // Slice of embeds to navigate
	ChannelID  string                    // Channel ID where the slider is sent
	CurrentIdx int                       // Current index in the embeds slice
	Session    *discordgo.Session        // Discord session
	MessageID  string                    // Message ID of the sent slider message
	Timeout    int                       // Timeout in seconds
}

// Disable updates the slider's buttons to be disabled
func (s *Slider) Disable() {
	disabledComponents := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Previous",
					Style:    discordgo.PrimaryButton,
					CustomID: "prev",
					Disabled: true,
				},
				discordgo.Button{
					Label:    "Next",
					Style:    discordgo.PrimaryButton,
					CustomID: "next",
					Disabled: true,
				},
			},
		},
	}

	_, err := s.Session.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         s.MessageID,
		Channel:    s.ChannelID,
		Components: &disabledComponents,
	})
	if err != nil {
		log.Trace("Error disabling slider: %v", err)
	}

	// Remove slider from the map
	sliders.Delete(s.MessageID)
}

// Modified Send method
func (s *Slider) Send() error {
	components := s.createButtons()

	msg, err := s.Session.ChannelMessageSendComplex(s.ChannelID, &discordgo.MessageSend{
		Embed:      s.Embeds[s.CurrentIdx],
		Components: components,
	})
	if err != nil {
		return err
	}

	s.MessageID = msg.ID
	sliders.Store(msg.ID, s)

	// Start a goroutine to disable the slider after the timeout
	go func() {
		if s.Timeout > 0 {
			<-time.After(time.Duration(s.Timeout) * time.Second)
			s.Disable()
		}
	}()

	return nil
}

// NewSlider initializes a Slider instance
func NewSlider(session *discordgo.Session, channelID string, embeds []*discordgo.MessageEmbed) (*Slider, error) {
	if len(embeds) == 0 {
		log.Error("embeds slice cannot be empty")
		return nil, fmt.Errorf("embeds slice cannot be empty")
	}
	return &Slider{
		Embeds:     embeds,
		ChannelID:  channelID,
		CurrentIdx: 0,
		Session:    session,
		Timeout:    300,
	}, nil
}

// HandleInteraction updates the embed based on interaction inputs
func HandleInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Retrieve the slider instance from the map
	if sliderInterface, ok := sliders.Load(i.Message.ID); ok {
		slider := sliderInterface.(*Slider)

		// Update the current index based on the button clicked
		switch i.MessageComponentData().CustomID {
		case "prev":
			if slider.CurrentIdx > 0 {
				slider.CurrentIdx--
			}
		case "next":
			if slider.CurrentIdx < len(slider.Embeds)-1 {
				slider.CurrentIdx++
			}
		}

		// Update the embed and components
		err := slider.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{slider.Embeds[slider.CurrentIdx]},
				Components: slider.createButtons(),
			},
		})
		if err != nil {
			log.Trace("Error updating slider: %v", err)
		}
	}
}

// createButtons generates the navigation buttons for the slider
func (s *Slider) createButtons() []discordgo.MessageComponent {
	prevDisabled := s.CurrentIdx == 0
	nextDisabled := s.CurrentIdx == len(s.Embeds)-1

	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Previous",
					Style:    discordgo.PrimaryButton,
					CustomID: "prev",
					Disabled: prevDisabled,
				},
				discordgo.Button{
					Label:    "Next",
					Style:    discordgo.PrimaryButton,
					CustomID: "next",
					Disabled: nextDisabled,
				},
			},
		},
	}
}

func CreateFoodEmbed(info FoodInfo) (*discordgo.MessageEmbed, error) {

	var builder strings.Builder
	for key, value := range info.NutritionFacts {
		builder.WriteString(fmt.Sprintf(" - %s: %s\n", key, value))
	}
	nutritionFacts := builder.String()
	builder.Reset()

	builder.WriteString(fmt.Sprintf("- %s **Room Temperature Safety Window:** %s days\n", storageEmoji("room_temp"), info.RoomTemp.FoodSafetyWindow))
	builder.WriteString(fmt.Sprintf("- %s **Room Temperature Expected Expiration:** %s \n", storageEmoji("room_temp"), info.RoomTemp.Expiration))
	builder.WriteString(fmt.Sprintf("- %s **Fridge Safety Window:** %s days\n", storageEmoji("fridge"), info.Fridge.FoodSafetyWindow))
	builder.WriteString(fmt.Sprintf("- %s **Fridge Expected Expiration:** %s\n", storageEmoji("fridge"), info.Fridge.Expiration))
	storage_info := builder.String()
	embed := discordgo.MessageEmbed{
		Color:       0x00ff00, // green,
		Description: fmt.Sprintf("%s %s info", info.FoodEmoji, info.FoodItem),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Nutrition Facts",
				Value:  nutritionFacts,
				Inline: true,
			},
			{
				Name:   "Recommended Storage",
				Value:  fmt.Sprintf("%s %s", info.Storage, storageEmoji(info.Storage)),
				Inline: true,
			},
			{
				Name:   "Storage Information",
				Value:  storage_info,
				Inline: false,
			},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: info.ImageUrl,
		},
		Title: "Food Information",
	}
	return &embed, nil

}

func CreateRecipeEmbed(recipe Recipe) (*discordgo.MessageEmbed, error) {

	var builder strings.Builder
	for _, nutrient := range recipe.MacroNutrients {
		builder.WriteString(fmt.Sprintf(" - %s\n", nutrient))
	}
	nutritionFacts := builder.String()
	builder.Reset()

	for _, ingredient := range recipe.Ingredients {
		builder.WriteString(fmt.Sprintf(" - %s\n", ingredient))
	}
	allIngredients := builder.String()

	builder.Reset()
	for _, ingredient := range recipe.MissingIngredients {
		builder.WriteString(fmt.Sprintf(" - %s\n", ingredient))
	}
	missingIngredients := builder.String()
	builder.Reset()

	embed := discordgo.MessageEmbed{
		Color:       0x00ff00, // green,
		Description: recipe.Description,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   fmt.Sprintf("🍽️ Serves %s | Ingredients", recipe.NumberOfServings),
				Value:  allIngredients,
				Inline: true,
			},
			{
				Name:   "Missing ingredients",
				Value:  missingIngredients,
				Inline: true,
			},
			{
				Name:   "Cooking Instructions",
				Value:  recipe.CookingInstructions,
				Inline: false,
			},
			{
				Name:   "Additional seasoning",
				Value:  recipe.AdditionalSeasoning,
				Inline: false,
			},
			{
				Name:   "Macronutrients",
				Value:  nutritionFacts,
				Inline: false,
			},
		},
		Thumbnail: &discordgo.MessageEmbedThumbnail{
			URL: "https://media.discordapp.net/attachments/740596566632562763/1325175644346388530/file-WF56sz777XLvWwNMFRGaBs.png?ex=677ad57e&is=677983fe&hm=dc54f362fa2f7c1efb4b54b7fc2b515a7061487c531b33538763d47e901123bd&=&format=webp&quality=lossless&width=840&height=840",
		},
		Title: fmt.Sprintf("Recipe: %s", recipe.RecipeName),
	}
	return &embed, nil

}
