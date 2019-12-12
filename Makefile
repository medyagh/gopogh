build:
	go build -o gopogh

generate_json:
	go tool test2json -t < ./testdata/minikube-logs.txt > ./testdata/minikube-logs.json

test: build
	rm output.html || true
	./gopogh -in testdata/minikube-logs.json -out output.html
