FROM ubuntu:latest

WORKDIR /revtr
COPY . /revtr

RUN useradd -ms /bin/bash revtr

USER revtr

ENTRYPOINT ["/revtr/revtr"]
CMD ["-loglevel", "error"]

EXPOSE 8080
