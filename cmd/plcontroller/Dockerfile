FROM golang

ADD . /go/src/github.com/NEU-SNS/ReverseTraceroute

WORKDIR /go/src/github.com/NEU-SNS/ReverseTraceroute
RUN go get github.com/tools/godep
RUN godep restore ./...
RUN go install github.com/NEU-SNS/ReverseTraceroute/cmd/plcontroller

ENTRYPOINT /go/bin/plcontroller -loglevel fatal

EXPOSE 4380
EXPOSE 4381

