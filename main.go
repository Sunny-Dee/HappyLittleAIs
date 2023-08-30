package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"happylittleais/config"
	imgGen "happylittleais/image_generator"
	"happylittleais/social"
)

// TODO refresh IG token as part of flow
// TODO return named values
// TODO clean up naming 
// TODO unit tests
const timeout = 5 * time.Minute

func main() {
	configuration, err := config.LoadConfig(".")
	if err != nil {
		log.Printf("cannot load config from file. Error %v", err)
		configuration = config.LoadFromEnvVars()
		log.Println("loading config from environment variables")
		if len(configuration.ChatGptToken) == 0 || len(configuration.IgID) == 0 || len(configuration.IgToken) == 0 {
			log.Panic("Env vars not set.")
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	prompt, err := imgGen.GeneratePrompt(ctx, configuration.ChatGptToken)
	if err != nil {
		log.Panicf("Error generating prompt. %v", err)
	}

	log.Printf("Prompt:\n%s\n", prompt)

	imgUrl, err := imgGen.GenerateImage(ctx, fmt.Sprintf("Digital art, %s", prompt), configuration.ChatGptToken)
	if err != nil {
		log.Panicf("Generating image failed. %v", err)
	}
	log.Printf("Encoded image URL:\n%s\n", imgUrl)

	caption := fmt.Sprintf("Prompt: %s", prompt)
	mediaID, err := social.CreateMedia(ctx, imgUrl, caption, configuration.IgToken, configuration.IgID)
	if err != nil {
		log.Panicf("Media not created. Error: %v", err)
	}

	log.Printf("Media ID: %s", mediaID)
	err = social.PostImage(ctx, mediaID, configuration.IgToken, configuration.IgID)
	if err != nil {
		log.Panicf("Could not post image. Error %v\n", err)
	}

	fmt.Println("Image posted to Instagram.")
}
