package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func UnmarshalFood(jsonString string) (*FoodInfo, error) {
	var info FoodInfo
	err := json.Unmarshal([]byte(jsonString), &info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return &info, nil
}
func UnmarshalFoods(jsonString string) ([]FoodInfo, error) {
	var info []FoodInfo
	err := json.Unmarshal([]byte(jsonString), &info)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return info, nil
}

func UnmarshalRecipes(jsonString string) ([]Recipe, error) {
	var recipes []Recipe
	err := json.Unmarshal([]byte(jsonString), &recipes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	return recipes, nil
}

// PrettyMarkdownResponse generates a Markdown representation of the food information.
func PrettyMarkdownResponse(info FoodInfo) (string, error) {
	// Build the Markdown response
	var markdown bytes.Buffer
	markdown.WriteString("### Food Information\n\n")
	markdown.WriteString(fmt.Sprintf("- **Food Item:** %s %s\n", info.FoodItem, info.FoodEmoji))
	markdown.WriteString(fmt.Sprintf("- **Barcode Number:** %s\n\n", info.BarcodeNumber))

	markdown.WriteString("#### Nutrition Facts:\n")
	for key, value := range info.NutritionFacts {
		markdown.WriteString(fmt.Sprintf("- %s: %s\n", capitalizeFirstLetter(key), value))
	}
	markdown.WriteString("\n")

	markdown.WriteString("#### Storage Information:\n")
	markdown.WriteString(fmt.Sprintf("- **Preferred Storage:** %s %s\n", info.Storage, storageEmoji(info.Storage)))
	markdown.WriteString(fmt.Sprintf("- %s **Room Temperature Safety Window:** %s days\n", storageEmoji("room_temp"), info.RoomTemp.FoodSafetyWindow))
	markdown.WriteString(fmt.Sprintf("- %s **Room Temperature Expected Expiration:** %s \n", storageEmoji("room_temp"), info.RoomTemp.Expiration))
	markdown.WriteString(fmt.Sprintf("- %s **Fridge Safety Window:** %s days\n", storageEmoji("fridge"), info.Fridge.FoodSafetyWindow))
	markdown.WriteString(fmt.Sprintf("- %s **Fridge Expected Expiration:** %s\n", storageEmoji("fridge"), info.Fridge.Expiration))

	return markdown.String(), nil
}

// capitalizeFirstLetter capitalizes the first letter of a string.
func capitalizeFirstLetter(s string) string {
	if len(s) == 0 {
		return s
	}
	return fmt.Sprintf("%s%s", strings.ToUpper(string(s[0])), s[1:])
}

// storageEmoji returns an emoji based on the storage type.
func storageEmoji(storage string) string {
	switch strings.ToLower(storage) {
	case "fridge":
		return "❄️" // Snowflake for fridge
	case "room_temp":
		return "🏠" // House for room temperature
	default:
		return ""
	}
}
