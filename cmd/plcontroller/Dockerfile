FROM golang

ENV LD_LIBRARY_PATH=/lib:/usr/lib:/usr/local/lib
WORKDIR /go/src/github.com/NEU-SNS/ReverseTraceroute
ADD . /go/src/github.com/NEU-SNS/ReverseTraceroute


RUN useradd -ms /bin/bash plcontroller
RUN chown plcontroller:plcontroller id_rsa_pl
RUN mkdir -p scamper && cd scamper \
    && wget http://www.ccs.neu.edu/home/rhansen2/scamper.tar.gz \
    && tar xzf scamper.tar.gz && cd scamper-cvs-20150901 \
    && ./configure && make install
RUN go get ./...
RUN cd /go/src/github.com/NEU-SNS/ReverseTraceroute/cmd/plcontroller \
    && go build -ldflags \
    "-X main.ConfigPath=/go/src/github.com/NEU-SNS/ReverseTraceroute/cmd/plcontroller/plcontroller.config -X main.Version=0.0.1 -X main.Build=8" 

USER plcontroller

ENTRYPOINT ["cmd/plcontroller/plcontroller"]

EXPOSE 4380
EXPOSE 4381

