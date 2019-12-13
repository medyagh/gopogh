BINARY=gopogh
VERSION=`git fetch;git describe --tags > /dev/null 2>&1`
BUILD=`date +%FT%T%z`
LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.Build=${BUILD}"

generate_json:
	go tool test2json -t < ./testdata/minikube-logs.txt > ./testdata/minikube-logs.json

build: 
	CGO_ENABLED=0 go build ${LDFLAGS} -o ${BINARY}

.PHONY: test
test: build
	rm output.html || true
	./${BINARY} -in testdata/minikube-logs.json -out output.html

.PHONY: cross
cross: ${BINARY}-linux-amd64 ${BINARY}-darwin-amd64 ${BINARY}.exe

${BINARY}-darwin-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY}-darwin-amd64

${BINARY}-linux-amd64:
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY}-linux-amd64

${BINARY}.exe:
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY}.exe


.PHONY: clean
clean:
	rm ${BINARY}-linux-amd64 || true
	rm ${BINARY}-darwin-amd64 || true
	rm ${BINARY}.exe || true
