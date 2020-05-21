FROM golang:1.14-alpine AS build

WORKDIR /go/src/github.com/HackDalton/pretty-good-privacy
COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

FROM alpine

COPY --from=build /go/bin/pretty-good-privacy ./
COPY ./public ./public
COPY flag.txt flag.txt
COPY privatekey.asc privatekey.asc
COPY env.list  env.list

CMD ["./pretty-good-privacy"]