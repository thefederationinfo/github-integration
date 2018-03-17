FROM golang:1.9

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update
RUN apt-get install -y git-core
RUN apt-get clean && apt-get autoclean

RUN useradd -ms /bin/bash user

RUN git clone --depth 1 https://github.com/thefederationinfo/github-integration.git \
  $GOPATH/src/github.com/thefederationinfo/github-integration
WORKDIR $GOPATH/src/github.com/thefederationinfo/github-integration

RUN go get ./...
RUN go build
RUN mv github-integration /usr/local/bin/github-integration

RUN apt-get purge -y git-core
RUN apt-get clean && apt-get autoclean
RUN rm -rv $GOPATH/src/*

USER user
WORKDIR /home/user

EXPOSE 8181

CMD ["sh", "-c", "/usr/local/bin/github-integration --github-id ${GITHUB_ID} --github-secret ${GITHUB_SECRET} --server-domain ${DOMAIN} --travis-token ${TRAVIS_SECRET}"]
