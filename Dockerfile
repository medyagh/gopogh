FROM golang:alpine3.10
WORKDIR /src/github.com/medyah/gopogh
COPY ./ ./
RUN go get github.com/GeertJohan/go.rice
RUN go get github.com/GeertJohan/go.rice/rice
RUN apk -U add make
RUN make build
RUN cp ./gopogh /

