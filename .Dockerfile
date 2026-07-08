FROM golang:1.26 AS builder

WORKDIR /usr/src/app

COPY go.mod ./
RUN go mod download


COPY --exclude=go.* . .
RUN GOOS=linux GOARCH=amd64 go build -o reminder_bot main.go

FROM gcr.io/distroless/base-debian12:nonroot

COPY --from=builder --chown=nonroot:nonroot --chmod=755 /usr/src/app/reminder_bot .

ENTRYPOINT [ "./reminder_bot" ]