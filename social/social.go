package social

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
)

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

func CreateMedia(ctx context.Context, imgUrl, caption, igToken, igID string) (string, error) {
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

func PostImage(ctx context.Context, mediaID, igToken, igID string) error {
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