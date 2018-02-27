FROM debian:jessie

ENV DEBIAN_FRONTEND noninteractive

RUN apt-get update
RUN apt-get install -y curl

RUN useradd -ms /bin/bash user

RUN curl -o /usr/local/bin/github-integration -L https://github.com/thefederationinfo/github-integration/releases/download/v1.0.2/github-integration.$(uname -m)

RUN apt-get purge -y curl
RUN apt-get autoremove -y
RUN apt-get clean && apt-get autoclean

RUN chmod +x /usr/local/bin/github-integration

USER user
WORKDIR /home/user

EXPOSE 8181

CMD ["sh", "-c", "/usr/local/bin/github-integration --github-id ${GITHUB_ID} --github-secret ${GITHUB_SECRET} --server-domain ${DOMAIN} --travis-token ${TRAVIS_SECRET}"]
