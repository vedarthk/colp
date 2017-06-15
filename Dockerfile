FROM golang:1.8.3

RUN curl https://glide.sh/get | sh

COPY . /go/src/github.com/vedarthk/colp

WORKDIR /go/src/github.com/vedarthk/colp

RUN glide --debug install && go-wrapper install

ENTRYPOINT ["colp"]
