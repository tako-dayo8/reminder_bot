FROM golang:1.26.5 AS builder

WORKDIR /usr/src/app

COPY go.mod go.sum ./
RUN go mod download


COPY . .
RUN GOOS=linux GOARCH=amd64 go build -o reminder_bot main.go

FROM gcr.io/distroless/base-debian12:nonroot

WORKDIR /app
COPY --from=builder --chown=nonroot:nonroot --chmod=755 /usr/src/app/reminder_bot .

ENTRYPOINT [ "./reminder_bot" ]