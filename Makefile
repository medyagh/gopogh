BINARY=/out/gopogh
GIT_TAG=`git fetch;git describe --tags > /dev/null 2>&1`
COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
BUILD ?= $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")
LDFLAGS :=-X github.com/medyagh/gopogh/pkg/report.Build=${BUILD}

CMD_SOURCE_DIRS = cmd pkg
SOURCE_FILES = $(shell find $(CMD_SOURCE_DIRS) -type f -name "*.go" | grep -v _test.go)

.PHONY: embed-static
embed-static: # update this before each build. to embed template files into golang
	cd pkg/report && rice embed-go

.PHONY: build
build: out/gopogh

out/gopogh: embed-static $(SOURCE_FILES) go.mod
	CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -a -o $@ github.com/medyagh/gopogh/cmd/gopogh

out/gopogh-darwin-amd64: embed-static $(SOURCE_FILES) go.mod
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -a -o $@ github.com/medyagh/gopogh/cmd/gopogh

out/gopogh-linux-amd64: embed-static $(SOURCE_FILES) go.mod
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -a -o $@ github.com/medyagh/gopogh/cmd/gopogh

out/gopogh.exe: embed-static $(SOURCE_FILES) go.mod
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64  go build -ldflags="$(LDFLAGS)" -a -o $@ github.com/medyagh/gopogh/cmd/gopogh

# gopogh requires a json input, uses go tool test2json to convert to json
generate_json:
	go tool test2json -t < ./testdata/minikube-logs.txt > ./testdata/minikube-logs.json

.PHONY: test
test: build
	rm output.html || true
	.${BINARY} -name "KVM Linux" -repo "github.com/kubernetes/minikube/" -pr "6096" -in "testdata/minikube-logs.json" -out "./out/output.html" -details "0c07e808219403a7241ee5a0fc6a85a897594339"

.PHONY: cross
cross: out/gopogh-linux-amd64 out/gopogh-darwin-amd64 out/gopogh.exe


.PHONY: clean
clean:
	rm -rf out
	rm pkg/report/rice-box.go || true



.PHONY: build-image
build-image:
	docker build -t local/gopogh:latest .

.PHONY: test-in-docker
test-in-docker:
	rm ./testdata/docker-test/testout.json || true
	rm ./testdata/docker-test/testout.html || true
	docker run  -it -e NAME="${JOB_NAME} ${GITHUB_REF}" -e REPO="${GITHUB_REPOSITORY}" -e DETAILS="${GITHUB_SHA}" -v $(CURDIR)/testdata/docker-test:/data  local/gopogh ./text2html.sh