FROM golang:1.24.0-alpine
WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download
COPY . .

RUN go build -o /notify-controller

CMD [ "/notify-controller" ]
