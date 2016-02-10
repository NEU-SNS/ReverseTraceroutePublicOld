FROM ubuntu:latest

WORKDIR /atlas
COPY . /atlas

RUN useradd -ms /bin/bash atlas

USER atlas

ENTRYPOINT ["/atlas/atlas"]
CMD ["-loglevel", "error"]

EXPOSE 55000
EXPOSE 8080
