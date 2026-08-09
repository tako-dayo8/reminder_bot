FROM golang:1.26.5

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download


COPY ./cmd ./cmd
CMD [ "go", "run", "./cmd/main.go" ]