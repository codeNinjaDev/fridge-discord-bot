package main

import (
	"fmt"
	_ "image/jpeg" // Support for JPEG
	_ "image/png"  // Support for PNG
	"io"
	"net/http"
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
