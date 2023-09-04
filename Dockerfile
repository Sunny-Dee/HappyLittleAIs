# syntax=docker/dockerfile:1

FROM golang:1.19

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./
COPY image_generator/image_generator.go ./image_generator/
COPY social/social.go ./social/
COPY config/config.go ./config/
RUN CGO_ENABLED=0 GOOS=linux go build -o /happylittleais

CMD ["/happylittleais"]
