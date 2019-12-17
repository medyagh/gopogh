set +x
TEST_OUT=$(pwd)/minikube-logs.txt
JSON_OUT=$(pwd)/minikube-logs.json
touch ${JSON_OUT}
docker run --mount type=bind,source="${JSON_OUT}",target=/tmp/out.json \
           --mount type=bind,source="${TEST_OUT}",target=/tmp/log.txt \
           -i medyagh/gopogh:v0.0.8 \
           sh -c "go tool test2json -t < /tmp/log.txt > /tmp/out.json" 

HTML_OUT=$(pwd)/minikube-logs.html
touch ${HTML_OUT}

docker run --rm --mount type=bind,source=${JSON_OUT},target=/tmp/log.json \
                --mount type=bind,source="${HTML_OUT}",target=/tmp/log.html \
                -i medyagh/gopogh:v0.0.12 sh -c "/gopogh -in /tmp/log.json -out /tmp/log.html -name KVM linux -pr 6081 -repo github.com/kubernetes/minikube"


