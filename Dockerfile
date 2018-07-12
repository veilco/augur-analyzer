FROM golang:1.10.3-stretch

WORKDIR /go/src/github.com/stateshape/predictions.global/server

COPY . ./

RUN go build -o /bin/auguranalyzer main.go

CMD /bin/auguranalyzer
