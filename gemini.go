package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
)

func generateFoodString(ctx context.Context, model *genai.GenerativeModel, imgUrl string, filext string) (string, error) {
	model.ResponseMIMEType = "application/json"
	bytes, err := downloadImageToByteArray(imgUrl)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`Today's date: %v, From the provided image: 
		1. Identify the food item most relevant for a nutritionist.
		2. Provide the barcode number.
		3. List the likely nutrition facts (only include the calories and other macronutrients) with exact figures and units.
		4. Specify the most common storage method (choose between "room_temp" or "fridge").
		5. Estimate the food safety window (in days) and estimated food expiration date for both "room_temp" and "fridge".
		6. Return the output in the following JSON schema:
		---

		### JSON Schema
		{
		"food_item": "string",
		"barcode_number": "string", # int formatted as string
		"nutrition_facts": { 
			...
		},
		"storage": "string (room_temp or fridge)",
		"room_temp": {
			"food_safety_window": "number (days)" # int formatted as string
			"expected_expiration_date": "date" # maximum expiration in MM/DD/YY starting from tomorrows date when stored at room temperature
		},
		"fridge": {
			"food_safety_window": "number (days)" # int formatted as string
			"expected_expiration_date": "date" # maximum expiration in MM/DD/YY starting from tomorrows date when stored in fridge
		},
		"food_emoji": "emoji representing food item",
		"cost": "estimated price in dollars and cents" # e.g $D.CC
		}`, time.Now())

	req := []genai.Part{
		genai.ImageData(strings.TrimPrefix(".", filext), bytes),
		genai.Text(prompt),
	}

	resp, err := model.GenerateContent(ctx, req...)

	if err != nil {
		return "", err
	}

	// Handle the response of generated text
	if len(resp.Candidates) == 0 {
		return "", err
	}
	c := resp.Candidates[0]
	var builder strings.Builder
	for _, part := range c.Content.Parts {
		builder.WriteString(fmt.Sprint(part))
	}
	// Trim any trailing space and print the result
	result := strings.TrimSpace(builder.String())
	return result, err
}

func generateAllFoodStringsFromReceipt(ctx context.Context, model *genai.GenerativeModel, imgUrl string, filext string) (string, error) {
	model.ResponseMIMEType = "application/json"
	bytes, err := downloadImageToByteArray(imgUrl)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`Today's date: %v, 
		The image should be a receipt from the store.
		For all the foods that are listed in the receipt:

		1. Identify the full name of the food item most relevant for a nutritionist.
		3. List the likely nutrition facts (only include the calories and other macronutrients) with exact figures and units.
		4. Specify the most common storage method (choose between "room_temp" or "fridge").
		5. Estimate the food safety window (in days) and estimated food expiration date for both "room_temp" and "fridge".
		6. Return the output in the following JSON schema:
		---

		### JSON Schema
		[{
		"food_item": "string",
		"nutrition_facts": { 
			...
		},
		"storage": "string (room_temp or fridge)",
		"room_temp": {
			"food_safety_window": "number (days)" # int formatted as string
			"expected_expiration_date": "date" # maximum expiration in MM/DD/YY starting from tomorrows date when stored at room temperature
		},
		"fridge": {
			"food_safety_window": "number (days)" # int formatted as string
			"expected_expiration_date": "date" # maximum expiration in MM/DD/YY starting from tomorrows date when stored in fridge
		},
		"food_emoji": "emoji representing food item",
		"cost": "estimated price in dollars and cents" # e.g $D.CC
		}, ...]`, time.Now())

	req := []genai.Part{
		genai.ImageData(strings.TrimPrefix(".", filext), bytes),
		genai.Text(prompt),
	}

	resp, err := model.GenerateContent(ctx, req...)

	if err != nil {
		return "", err
	}

	// Handle the response of generated text
	if len(resp.Candidates) == 0 {
		return "", err
	}
	c := resp.Candidates[0]
	var builder strings.Builder
	for _, part := range c.Content.Parts {
		builder.WriteString(fmt.Sprint(part))
	}
	// Trim any trailing space and print the result
	result := strings.TrimSpace(builder.String())
	return result, err
}

func generateAllFoodStrings(ctx context.Context, model *genai.GenerativeModel, imgUrl string, filext string, photo bool) (string, error) {
	if photo {
		return generateFoodStringsFromPhoto(ctx, model, imgUrl, filext)
	} else {
		return generateAllFoodStringsFromReceipt(ctx, model, imgUrl, filext)
	}
}
func generateFoodStringsFromPhoto(ctx context.Context, model *genai.GenerativeModel, imgUrl string, filext string) (string, error) {
	model.ResponseMIMEType = "application/json"
	bytes, err := downloadImageToByteArray(imgUrl)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`Today's date: %v, 
		For all foods in the image:

		1. Identify the food item most relevant for a nutritionist.
		2. Provide the barcode number.
		3. List the likely nutrition facts (only include the calories and other macronutrients) with exact figures and units.
		4. Specify the most common storage method (choose between "room_temp" or "fridge").
		5. Estimate the food safety window (in days) and estimated food expiration date for both "room_temp" and "fridge".
		6. Return the output in the following JSON schema:
		---

		### JSON Schema
		[{
		"food_item": "string",
		"barcode_number": "string", # int formatted as string
		"nutrition_facts": { 
			...
		},
		"storage": "string (room_temp or fridge)",
		"room_temp": {
			"food_safety_window": "number (days)" # int formatted as string
			"expected_expiration_date": "date" # maximum expiration in MM/DD/YY starting from tomorrows date when stored at room temperature
		},
		"fridge": {
			"food_safety_window": "number (days)" # int formatted as string
			"expected_expiration_date": "date" # maximum expiration in MM/DD/YY starting from tomorrows date when stored in fridge
		},
		"food_emoji": "emoji representing food item",
		"cost": "estimated price in dollars and cents" # e.g $D.CC
		}, ...]`, time.Now())

	req := []genai.Part{
		genai.ImageData(strings.TrimPrefix(".", filext), bytes),
		genai.Text(prompt),
	}

	resp, err := model.GenerateContent(ctx, req...)

	if err != nil {
		return "", err
	}

	// Handle the response of generated text
	if len(resp.Candidates) == 0 {
		return "", err
	}
	c := resp.Candidates[0]
	var builder strings.Builder
	for _, part := range c.Content.Parts {
		builder.WriteString(fmt.Sprint(part))
	}
	// Trim any trailing space and print the result
	result := strings.TrimSpace(builder.String())
	return result, err
}

func generateRecipes(ctx context.Context, model *genai.GenerativeModel, foods []FoodInfo, additional_prompt string) (string, error) {
	model.ResponseMIMEType = "application/json"

	var builder strings.Builder
	builder.WriteString("[")
	for _, food := range foods {
		builder.WriteString(food.FoodItem)
		builder.WriteString(",")
	}
	builder.WriteString("]")

	prompt := fmt.Sprintf(`
	I have the following foods in my fridge or pantry:
	%s

	Please recommend 3 recipes that primarily use ingredients from this list. The recipes should:

	1. Strongly prioritize using ingredients from the list.
	2. Allow for occasional deviations by including a few ingredients not on the list, but only if they are essential to complete the dish and are common or easy to substitute (e.g., butter, garlic, or spices) OR if it is necessary to fulfill the user's special preferences.
	3. Provide the name of the dish, the required ingredients, and a brief preparation method.

	If you include ingredients not on the list, please clearly indicate them and suggest potential substitutions using items I might already have. Feel free to suggest creative or healthier variations for each recipe. If the user preferences are not empty, each recipe should strongly relate to the user's preference. The user added these preferences: %s

	Output in the following JSON schema:

	[{
		"recipe_name": "string",
		"description": "string",
		"number_of_servings": "string", # int formatted as string
		"ingredients": ["string1", "string2", ...], # each ingredient and emoji e.g 2 tbsp of olive oil 🫒 (includes all ingredients)
		"missing_ingredients": [] # list of each ingredient missing from fridge or pantry
		"cooking_instructions": "string", # markdown formatted step-by-step preparation and cooking instructions make sure to use new line characters
		"additonal_seasoning": "string", # markdown formatted suggestions for additional seasoning of dish
		"macro_nutrients": [] # a list of strings containing the macronutrient content (per serving) for the dish in grams in a way that would be useful to a nutritionist
		"cost": "estimated price in dollars and cents" # e.g $D.CC
	},...]`, builder.String(), additional_prompt)

	req := []genai.Part{
		genai.Text(prompt),
	}

	resp, err := model.GenerateContent(ctx, req...)

	if err != nil {
		return "", err
	}

	// Handle the response of generated text
	if len(resp.Candidates) == 0 {
		return "", err
	}
	c := resp.Candidates[0]
	builder.Reset()
	for _, part := range c.Content.Parts {
		builder.WriteString(fmt.Sprint(part))
	}
	// Trim any trailing space and print the result
	result := strings.TrimSpace(builder.String())
	return result, err
}
