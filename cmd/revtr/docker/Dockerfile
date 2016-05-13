FROM ubuntu:latest

WORKDIR /revtr
COPY . /revtr

RUN apt-get update
RUN apt-get install -y ca-certificates &&  apt-get clean \
&&  rm -rf /var/lib/apt/lists/*

ENTRYPOINT ["/revtr/revtr"]
CMD ["-loglevel", "error"]

EXPOSE 8080
EXPOSE 8181
EXPOSE 45454