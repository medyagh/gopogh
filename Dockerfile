FROM golang:1.24-alpine
WORKDIR /src/
COPY ./ ./
RUN apk -U add make git
RUN make build
RUN install ./out/gopogh /bin/gopogh
RUN chmod +x ./text2html.sh
RUN cp ./text2html.sh /text2html.sh


