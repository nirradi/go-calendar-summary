FROM golang:1.12

WORKDIR /go/src/app
COPY ./src .

RUN go get
RUN go build -o /go-api-demo *.go
COPY run.sh /usr/local/bin

CMD ["run.sh"]