#!/bin/bash

TEST_INPUT="$(pwd)/testdata/minikube-logs2.txt"
TEST_OUT="$(pwd)/testdata/minikube-logs2.out"
JSON_OUT="$(pwd)/testdata/minikube-logs2.json"
HTML_OUT="$(pwd)/testdata/minikube-logs2.html"
echo $TEST_INPUT
rm ${JSON_OUT} > /dev/null 2>&1 || true
touch "${JSON_OUT}"
rm ${TEST_OUT} > /dev/null 2>&1 || true
touch ${TEST_OUT}
rm ${HTML_OUT} > /dev/null 2>&1 || true 
touch ${HTML_OUT}
JOB_NAME=VirtualBox_Linux
DOCKER_BIN=docker
MINIKUBE_LOCATION=6081
COMMIT=cd7cac61d3f8df1026f6b4a2689b362e132dfe4b

${DOCKER_BIN} run --mount type=bind,source="${JSON_OUT}",target=/tmp/out.json \
           --mount type=bind,source="${TEST_INPUT}",target=/tmp/log.txt \
           -i medyagh/gopogh:v0.0.13 \
           sh -c "go tool test2json -t < /tmp/log.txt > /tmp/out.json" || true


set +ex
${DOCKER_BIN} run --rm --mount type=bind,source=${JSON_OUT},target=/tmp/log.json \
                --mount type=bind,source="${HTML_OUT}",target=/tmp/log.html \
                -i medyagh/gopogh:v0.0.13 sh -c \
                "/gopogh -in /tmp/log.json -out /tmp/log.html  -name "${JOB_NAME}" -pr ${MINIKUBE_LOCATION} -repo github.com/kubernetes/minikube/  -details ${COMMIT}" || true
