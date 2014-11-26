FROM centos:centos7

RUN yum install -y go git

COPY . /kanikousen
WORKDIR /kanikousen

# Install go dependencies
RUN mkdir -p pkg
ENV GOPATH /kanikousen/pkg
# TODO what a ugly way
RUN go build src/main.go 2>&1 | perl -ne '/find package "([^"]+)/ and print "$1\n"' | xargs go get

ENV GOPATH /kanikousen/pkg:/kanikousen/src
RUN go build -o kanikousen src/main.go

ENTRYPOINT ["/kanikousen/kanikousen"]
