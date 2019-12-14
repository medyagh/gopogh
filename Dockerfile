FROM golang:alpine3.10
WORKDIR /src/github.com/medyah/gopogh
COPY ./ ./
RUN apk -U add make
RUN make build
RUN cp ./gopogh /

