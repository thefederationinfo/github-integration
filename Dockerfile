FROM debian:jessie

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update
RUN apt-get install -y curl
RUN apt-get clean && apt-get autoclean

RUN useradd -ms /bin/bash user

USER user

WORKDIR /home/user
RUN curl -o /home/user/github-integration -L https://github.com/thefederationinfo/github-integration/releases/download/v1.0.1/github-integration.$(uname -m)

RUN chmod +x /home/user/github-integration

EXPOSE 8181

CMD ["sh", "-c", "/home/user/github-integration --github-id ${GITHUB_ID} --github-secret ${GITHUB_SECRET} --server-domain ${DOMAIN} --travis-token ${TRAVIS_SECRET}"]
