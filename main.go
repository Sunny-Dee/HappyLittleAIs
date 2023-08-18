package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// TODO move functions to their own files or packages
// TODO refresh IG token as part of flow
// TODO return named values
const timeout = 5 * time.Minute

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

type MediaRequest struct {
	ImgURL string `json:"image_url"`
	Caption string `json:"caption"`
}

type MediaResponse struct {
	ID string `json:"id"`
}

type PublishRequest struct {
	Token string `url:"access_token"`
	ID string `url:"creation_id"`
}

func main() {
	config, err := loadConfig(".")
	if err != nil {
		log.Printf("cannot load config from file. Error %v", err)
		config = loadEnvVars()
		log.Println("loading config from environment variables")
		if len(config.ChatGptToken) == 0 || len(config.IgID) == 0 || len(config.IgToken) == 0 {
			log.Panic("Env vars not set.")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	prompt, err := generatePrompt(ctx, config.ChatGptToken)
	if err != nil {
		log.Panicf("Error generating prompt. %v", err)
	}

	log.Printf("Prompt:\n%s\n", prompt)

	imgUrl, err := generateImage(ctx, fmt.Sprintf("Digital art, %s", prompt), config.ChatGptToken)
	if err != nil {
		log.Panicf("Generating image failed. %v", err)
	}
	log.Printf("Encoded image URL:\n%s\n", imgUrl)

	caption := fmt.Sprintf("Prompt: %s", prompt)
 	mediaID, err := createMedia(ctx, imgUrl, caption, config.IgToken, config.IgID)
	if err != nil {
		log.Panicf("Media not created. Error: %v", err)
	}

	log.Printf("Media ID: %s", mediaID)
	err = postImage(ctx, mediaID, config.IgToken, config.IgID)
	if err != nil {
		log.Panicf("Could not post image. Error %v\n", err)
	}

	fmt.Println("Image posted to Instagram.")
}

func generatePrompt(ctx context.Context, token string) (string, error) {
	client := &http.Client{}
	url := "https://api.openai.com/v1/chat/completions"

	// Create the JSON body of the request
	body := strings.NewReader(`{
		"model": "gpt-3.5-turbo",
		"temperature": 1,
		"messages": [
			{
				"role": "system",
				"content": "Original art piece description in a specific style in about 30 words."
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

func generateImage(ctx context.Context, prompt, token string) (string, error) {
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

func createMedia(ctx context.Context, imgUrl, caption, igToken, igID string) (string, error) {
	urlStr := fmt.Sprintf("https://graph.facebook.com/v15.0/%v/media?access_token=%s&image_url=%s&caption=%s", 
	igID, igToken, imgUrl, url.PathEscape(caption))

	request, err := http.NewRequestWithContext(ctx, "POST", urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("creating request. Error: %v", err)
	}
	request.Header.Add("cache-control", "no-cache")

	// Send the request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return "", fmt.Errorf("executing request. Error: %v", err)
	}

	// Read the response body
	data, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	
	// Check the status code
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("response code not 200 OK. Response code: %v\nData:\n%q", response.StatusCode, data)
	}
	
	// Parse to json
	jResponse := MediaResponse{}
	err = json.Unmarshal([]byte(data), &jResponse)

	if err != nil {
		return "", err
	}
	return jResponse.ID, nil
}

func postImage(ctx context.Context, mediaID, igToken, igID string) error {
	urlStr := fmt.Sprintf("https://graph.facebook.com/v15.0/%v/media_publish?access_token=%s&creation_id=%s", 
	igID, igToken, mediaID)

	publishRequest, err := json.Marshal(PublishRequest{
		//Token: igToken,
		ID: mediaID,
	})
	if err != nil {
		return fmt.Errorf("marshalling request. Error: %v", err)
	}

	request, err := http.NewRequestWithContext(ctx, "POST", urlStr, bytes.NewReader(publishRequest))
	if err != nil {
		return fmt.Errorf("creating request. Error: %v", err)
	}

	// Send the request
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("executing request. Error: %v", err)
	}

	// Check the status code
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Response code not 200 OK. Response code: %v", response.StatusCode)
	}

	// Read the response body
	data, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	
	log.Printf("Response: %s\n", data)
	return nil
}

type Config struct {
	ChatGptToken string `mapstructure:"CHAT_GPT_TOKEN"`
	IgToken string `mapstructure:"IG_TOKEN"`
	IgID string `mapstructure:"IG_ID"`
}
func loadConfig(path string)(config Config, err error) {

	viper.AddConfigPath(path)
	viper.SetConfigName("app")
	viper.SetConfigType("env")

	err = viper.ReadInConfig()
    if err != nil {
        return
    }

    err = viper.Unmarshal(&config)
    return
}

func loadEnvVars()Config {
	return Config {
		ChatGptToken: os.Getenv("CHAT_GPT_TOKEN"),
		IgToken: os.Getenv("IG_TOKEN"),
		IgID: os.Getenv("IG_ID"),
	}
}