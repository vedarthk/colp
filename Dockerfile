FROM golang:1.8.3

RUN curl https://glide.sh/get | sh

VOLUME /go/src/app

WORKDIR /code

COPY docker-entrypoint.sh /usr/bin/
RUN chmod +x /usr/bin/docker-entrypoint.sh

ENTRYPOINT ["docker-entrypoint.sh"]
