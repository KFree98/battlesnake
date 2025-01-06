FROM golang:1.18

WORKDIR /usr/src/app

COPY . .
RUN go mod download && go mod verify
RUN go build -v -o /usr/local/bin/app ./...

ENV PORT=8080

EXPOSE 8080

CMD ["app"]
