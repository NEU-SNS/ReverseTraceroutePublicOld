FROM ubuntu:latest

WORKDIR /plcontroller
COPY . /plcontroller

RUN useradd -ms /bin/bash plcontroller
RUN apt-get update && apt-get install -y \
    wget \
    build-essential \
&&  apt-get clean \
&&  rm -rf /var/lib/apt/lists/*

RUN mkdir -p scamper && cd scamper \
    && wget http://www.ccs.neu.edu/home/rhansen2/scamper.tar.gz \
    && tar xzf scamper.tar.gz && cd scamper-cvs-20150901 \
    && ./configure && make install


USER plcontroller

ENTRYPOINT ["/plcontroller/plcontroller"]
CMD ["-loglevel", "error"]

EXPOSE 4380
EXPOSE 4381
EXPOSE 55556
