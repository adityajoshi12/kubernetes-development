FROM golang:1.22rc2-alpine
WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download
COPY . .

RUN go build -o /notify-controller

CMD [ "/notify-controller" ]
