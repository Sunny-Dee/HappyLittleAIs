package image_generator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
)

type Response struct {
	Choices []Choice `json:"choices"`
}

type Choice struct {
	Message Message `json:"message"`
}

type Message struct {
	Content string `json:"content"`
}

type ImageRequest struct {
	Prompt    string `json:"prompt"`
	NumImages int    `json:"n"`
	Size      string `json:"size"`
}

type ImageResponse struct {
	Data []Data `json:"data"`
}

type Data struct {
	Url string `json:"url"`
}

func GeneratePrompt(ctx context.Context, token string) (string, error) {
	client := &http.Client{}
	url := "https://api.openai.com/v1/chat/completions"

	// Create the JSON body of the request
	body := strings.NewReader(`{
		"model": "gpt-3.5-turbo",
		"temperature": 1,
		"messages": [
			{
				"role": "system",
				"content": "in a few words, give me an art idea in a particular style"
			}
		]
	}`)

	// Create the request
	request, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Send the request
	response, err := client.Do(request)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	// Check the status code
	if response.StatusCode != http.StatusOK {
		fmt.Println(response.StatusCode)
		return "", err
	}

	// Read the response body
	data, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	// Parse to json
	jResponse := Response{}
	err = json.Unmarshal([]byte(data), &jResponse)

	if err != nil {
		fmt.Println(err)
		return "", err
	}
	if len(jResponse.Choices) == 0 {
		return "", fmt.Errorf("no choices returned in request response")
	}

	// We are only asking for one response
	prompt := jResponse.Choices[0].Message.Content

	return prompt, nil
}

func GenerateImage(ctx context.Context, prompt, token string) (string, error) {
	urlStr := "https://api.openai.com/v1/images/generations"

	imageRequest, err := json.Marshal(ImageRequest{
		Prompt:    prompt,
		NumImages: 1,
		Size:      "1024x1024",
	})
	if err != nil {
		return "", err
	}

	request, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(imageRequest))
	if err != nil {
		return "", fmt.Errorf("creating request. %v", err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// Send the request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("executing request. %v", err)
	}

	// Check the status code
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("response code not 200 OK. Code %d", response.StatusCode)
	}

	// Read the response body
	data, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	// Parse to json
	jResponse := ImageResponse{}
	err = json.Unmarshal([]byte(data), &jResponse)

	if err != nil {
		return "", err
	}

	if len(jResponse.Data) == 0 {
		return "", fmt.Errorf("Response data does not contain a url. \n%q", data)
	}
	imgUrl := jResponse.Data[0].Url
	log.Printf("Image URL:\n%s\n", imgUrl)
	return url.QueryEscape(strings.TrimSpace(imgUrl)), nil
}
