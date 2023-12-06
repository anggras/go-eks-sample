FROM golang:1.19

WORKDIR /app

COPY /app /app

RUN go build main.go

EXPOSE 8080

CMD ["/app/main"]
