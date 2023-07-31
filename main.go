package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// TODO remove unnecessary print statements from non main methods
// TODO move functions to their own files or packages
// TODO refresh IG token as part of flow
// TODO return named values
// TODO change all print statements to log
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
		fmt.Printf("cannot load config. Error %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	prompt, err := generatePrompt(ctx, config.ChatGptToken)
	if err != nil {
		fmt.Printf("Error generating prompt. %v", err)
		cancel()
		return
	}

	fmt.Printf("Prompt:\n%s\n", prompt)

	imgUrl, err := generateImage(ctx, prompt, config.ChatGptToken)
	if err != nil {
		fmt.Println(err)
		cancel()
		return
	}
	fmt.Printf("Encoded image URL:\n%s\n", imgUrl)

	caption := fmt.Sprintf("Prompt: %q", prompt)
 	mediaID, err := createMedia(ctx, imgUrl, caption, config.IgToken, config.IgID)
	if err != nil {
		fmt.Printf("Media not created. Error: %v", err)
		cancel()
		return
	}

	fmt.Printf("Media ID: %s", mediaID)
	err = postImage(ctx, mediaID, config.IgToken, config.IgID)
	if err != nil {
		fmt.Printf("Could not post image. Error %v\n", err)
		cancel()
		return
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
				"content": "Describe a new and original art piece in the style of an artist in 50 words or less."
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
	data, err := ioutil.ReadAll(response.Body)
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
	data, err := ioutil.ReadAll(response.Body)
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
	fmt.Printf("Image URL:\n%s\n", imgUrl)
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
	data, err := ioutil.ReadAll(response.Body)
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
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return err
	}
	
	fmt.Printf("Response: %s\n", data)
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
