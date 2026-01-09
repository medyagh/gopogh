BINARY=/out/gopogh
GIT_TAG=`git describe --tags`
COMMIT_NO := $(shell git rev-parse HEAD 2> /dev/null || true)
BUILD ?= $(if $(shell git status --porcelain --untracked-files=no),"${COMMIT_NO}-dirty","${COMMIT_NO}")
LDFLAGS :=-X github.com/medyagh/gopogh/pkg/report.Build=${BUILD} -X github.com/medyagh/gopogh/pkg/report.version=${GIT_TAG}
MK_REPO=github.com/kubernetes/minikube/
DUMMY_COMMIT_NUM=0c07e808219403a7241ee5a0fc6a85a897594339
DUMMY_COMMIT2_NUM=0168d63fc8c165681b1cad1801eadd6bbe2c8a5c

BUILD_GOPOGH := CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -a -o
GOPOGH_CMD := github.com/medyagh/gopogh/cmd/gopogh

.PHONY: build
build: out/gopogh

.PHONY: out/gopogh
out/gopogh:
	$(BUILD_GOPOGH) $@ $(GOPOGH_CMD)

.PHONY: out/gopogh-darwin-arm64
out/gopogh-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(BUILD_GOPOGH) $@ $(GOPOGH_CMD)

.PHONY: out/gopogh-darwin-amd64
out/gopogh-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(BUILD_GOPOGH) $@ $(GOPOGH_CMD)

.PHONY: out/gopogh-linux-amd64
out/gopogh-linux-amd64:
	GOOS=linux GOARCH=amd64 $(BUILD_GOPOGH) $@ $(GOPOGH_CMD)

.PHONY: out/gopogh-linux-arm
out/gopogh-linux-arm:
	GOOS=linux GOARCH=arm $(BUILD_GOPOGH) $@ $(GOPOGH_CMD)

.PHONY: out/gopogh-linux-arm64
out/gopogh-linux-arm64:
	GOOS=linux GOARCH=arm64 $(BUILD_GOPOGH) $@ $(GOPOGH_CMD)

.PHONY: out/gopogh.exe
out/gopogh.exe:
	GOOS=windows GOARCH=amd64 $(BUILD_GOPOGH) $@ $(GOPOGH_CMD)

GOPOGH_SERVER_CMD := github.com/medyagh/gopogh/cmd/gopogh-server

.PHONY: server
server: out/gopogh-server

.PHONY: out/gopogh-server
out/gopogh-server:
	$(BUILD_GOPOGH) $@ $(GOPOGH_SERVER_CMD)

.PHONY: out/gopogh-server-darwin-arm64
out/gopogh-server-darwin-arm64:
        GOOS=darwin GOARCH=arm64 $(BUILD_GOPOGH) $@ $(GOPOGH_SERVER_CMD)

.PHONY: out/gopogh-server-darwin-amd64
out/gopogh-server-darwin-amd64:
        GOOS=darwin GOARCH=amd64 $(BUILD_GOPOGH) $@ $(GOPOGH_SERVER_CMD)

.PHONY: out/gopogh-server-linux-amd64
out/gopogh-server-linux-amd64:
        GOOS=linux GOARCH=amd64 $(BUILD_GOPOGH) $@ $(GOPOGH_SERVER_CMD)

.PHONY: out/gopogh-server-linux-arm
out/gopogh-server-linux-arm:
        GOOS=linux GOARCH=arm $(BUILD_GOPOGH) $@ $(GOPOGH_SERVER_CMD)

.PHONY: out/gopogh-server-linux-arm64
out/gopogh-server-linux-arm64:
        GOOS=linux GOARCH=arm64 $(BUILD_GOPOGH) $@ $(GOPOGH_SERVER_CMD)

.PHONY: out/gopogh-server.exe
out/gopogh-server.exe:
        GOOS=windows GOARCH=amd64 $(BUILD_GOPOGH) $@ $(GOPOGH_SERVER_CMD)

# gopogh requires a json input, uses go tool test2json to convert to json
generate_json:
	go tool test2json -t < ./testdata/minikube-logs.txt > ./testdata/minikube-logs.json

.PHONY: test
test: clean build
	.${BINARY} -name "KVM Linux" -repo "${MK_REPO}" -pr "6096" -in "testdata/minikube-logs.json" -out_html "./out/output.html" -out_summary out/output_summary.json -details "${DUMMY_COMMIT_NUM}"
	.${BINARY} -name "KVM Linux" -repo "${MK_REPO}" -pr "6096" -in "testdata/Docker_Linux.json" -out_html "./out/output2.html" -out_summary out/output2_summary.json -details "${DUMMY_COMMIT_NUM}"
	.${BINARY} -name "KVM Linux" -repo "${MK_REPO}" -pr "6096" -in "testdata/Docker_Linux.json" -out_html "./out/output2NoSummary.html" -details "${DUMMY_COMMIT_NUM}"

.PHONY: testdb
testdb: export DB_BACKEND=sqlite
testdb: export DB_PATH=out/testdb/output_db.db
testdb: clean build
	.${BINARY} -name "KVM Linux" -repo "${MK_REPO}" -pr "6096" -in "testdata/minikube-logs.json" -out_html "./out/output.html" -out_summary out/output_summary.json -db_path out/testdb/output_sqlite_summary.db -details "${DUMMY_COMMIT_NUM}"
	.${BINARY} -name "KVM Linux" -repo "${MK_REPO}" -pr "6096" -in "testdata/Docker_Linux.json" -out_html "./out/output2.html" -out_summary out/output2_summary.json -db_path out/testdb/output2_sqlite_summary.db -details "${DUMMY_COMMIT_NUM}"
	.${BINARY} -name "KVM Linux" -repo "${MK_REPO}" -pr "6096" -in "testdata/Docker_Linux.json" -out_html "./out/output2NoDBPath.html" -details "${DUMMY_COMMIT_NUM}"
	.${BINARY} -name "Docker MacOS" -repo "${MK_REPO}" -pr "16569" -in "testdata/testdb/Docker_macOS.json" -out_html "./out/docker_macOS_output.html" -details "${DUMMY_COMMIT2_NUM}"
	.${BINARY} -name "KVM Linux containerd" -repo "${MK_REPO}" -pr "16569" -in "testdata/testdb/KVM_Linux_containerd.json" -out_html "./out/kvm_linux_containerd_output.html" -details "${DUMMY_COMMIT2_NUM}"
	.${BINARY} -name "QEMU MacOS" -repo "${MK_REPO}" -pr "16569" -in "testdata/testdb/QEMU_macOS.json" -out_html "./out/qemu_macos_output.html" -details "${DUMMY_COMMIT2_NUM}"

.PHONY: testpgdb
testpgdb: export DB_BACKEND=postgres
testpgdb: export DB_PATH='host=k8s-minikube:us-west1:flake-rate user=postgres dbname=flakedbdev password=${DB_PASS}'
testpgdb: clean build
	.${BINARY} -name "KVM Linux" -repo "${MK_REPO}" -pr "6096" -in "testdata/minikube-logs.json" -out_html "./out/output.html" -out_summary out/output_summary.json -details "${DUMMY_COMMIT_NUM}" -use_cloudsql
	.${BINARY} -name "Docker MacOS" -repo "${MK_REPO}" -pr "16569" -in "testdata/testdb/Docker_macOS.json" -out_html "./out/docker_macOS_output.html" -details "${DUMMY_COMMIT2_NUM}" -use_cloudsql
	.${BINARY} -name "KVM Linux containerd" -repo "${MK_REPO}" -pr "16569" -in "testdata/testdb/KVM_Linux_containerd.json" -out_html "./out/kvm_linux_containerd_output.html" -details "${DUMMY_COMMIT2_NUM}" -use_cloudsql
	.${BINARY} -name "QEMU MacOS" -repo "${MK_REPO}" -pr "16569" -in "testdata/testdb/QEMU_macOS.json" -out_html "./out/qemu_macos_output.html" -details "${DUMMY_COMMIT2_NUM}" -use_cloudsql


.PHONY: cross
cross: out/gopogh-linux-amd64 out/gopogh-darwin-amd64 out/gopogh-darwin-arm64 out/gopogh.exe out/gopogh-linux-arm64 out/gopogh-linux-arm

.PHONY: lint
lint:
	golangci-lint run

.PHONY: clean
clean:
	rm -rf out

.PHONY: build-image
build-image:
	docker build -t local/gopogh:latest .

.PHONY: test-in-docker
test-in-docker:
	rm ./testdata/docker-test/testout.json || true
	rm ./testdata/docker-test/testout.html || true
	docker run  -it -e NAME="${JOB_NAME} ${GITHUB_REF}" -e REPO="${GITHUB_REPOSITORY}" -e DETAILS="${GITHUB_SHA}" -v $(CURDIR)/testdata/docker-test:/data  local/gopogh ./text2html.sh

.PHONY: azure_blob_connection_string
azure_blob_connection_string: ## set this env export AZURE_STORAGE_CONNECTION_STRING=$(az storage account show-connection-string -n $AZ_STORAGE -g $AZ_RG --query connectionString -o tsv)
	az storage account show-connection-string -n ${AZ_STORAGE} -g ${AZ_RG} --query connectionString -o tsv

load-fake-db:
	./hack/fakedb.sh $(RECORD_PATH)
