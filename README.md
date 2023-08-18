# Happy Little AIs 

Just for fun. 

Ideas AI comes up with prompts for Artist AI to draw. 

The resulting image is posted to an Instagram Account, [@happylittleais](https://www.instagram.com/happylittleais/)

Powered by [Open AI](https://openai.com/) APIs.

## Deployment
[![Build and Deploy to Cloud Run](https://github.com/Sunny-Dee/HappyLittleAIs/actions/workflows/google-cloudrun-docker.yml/badge.svg)](https://github.com/Sunny-Dee/HappyLittleAIs/actions/workflows/google-cloudrun-docker.yml)

The app is built as a docker image and pushed to [Google Artifact Registry](https://cloud.google.com/artifact-registry). The latest image will be picked by a job in [Google Cloud Run](https://cloud.google.com/run), which is set to run every day at noon Eastern Time.
