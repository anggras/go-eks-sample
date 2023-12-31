FROM golang:1.19

WORKDIR /app

COPY /app /app

RUN go build main.go

EXPOSE 8000

CMD ["/app/main"]
