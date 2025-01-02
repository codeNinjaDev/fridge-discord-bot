package main

import (
	"context"
	"fmt"
	_ "image/jpeg" // Support for JPEG
	_ "image/png"  // Support for PNG
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/generative-ai-go/genai"
)

func downloadImageToByteArray(url string) ([]byte, error) {
	// Get the image data
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad response: %d %s", resp.StatusCode, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func generateContent(ctx context.Context, model *genai.GenerativeModel, imgUrl string, filext string) (string, error) {
	model.ResponseMIMEType = "application/json"
	bytes, err := downloadImageToByteArray(imgUrl)
	if err != nil {
		return "", err
	}

	prompt := fmt.Sprintf(`Today's date: %v, From the provided image: 
		1. Identify the food item most relevant for a nutritionist.
		2. Provide the barcode number.
		3. List the likely nutrition facts with exact figures.
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
