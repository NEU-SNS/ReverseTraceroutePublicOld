FROM ubuntu:latest

WORKDIR /vpservice
COPY . /vpservice

RUN useradd -ms /bin/bash vpservice

USER vpservice

ENTRYPOINT ["/vpservice/vpservice"]
CMD ["-loglevel", "error"]

EXPOSE 45000
EXPOSE 8080
