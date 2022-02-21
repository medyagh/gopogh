# What is Gopogh ?

Converts golang test's json restults to a foldable html !

Example test logs: [before](https://storage.googleapis.com/minikube-builds/logs/6258/VirtualBox_Linux.txt), [after](https://storage.googleapis.com/minikube-builds/logs/6258/VirtualBox_Linux.html) 


## Give it a try

- first install gopogh
```
        go install github.com/medyagh/gopogh/cmd/gopogh@latest
```

- run your integraiton test and convert it to json

```
        go tool test2json -t < ./your-test-logs.txt > ./your-test-log.json
```
- run gopogh on it

```
gopogh -in ./your-test-log.json -out_html ./report/testout.html -out_summary ./your-test-summary.sjon TEST_NAME TEST_PR_NUMBER -repo "${GITHUB_REPOSITORY}"  -details "${GITHUB_SHA}")  
```

## Features:
- foldable test results.
- open each subt test restul in a new window
- sort test by passed/failed/skipped.
- sort test by execution duraiton.
- search in each test result separately.
- summary table
- gennerate json summary


## History 
I lead minikube team's, due to growing number of integration tests on multiple OS, drivers, container runtimes.
Each test failure on a PR generated  tens of thousands of lines for raw logs. (with system-level postmoretems)
that made reviewing PRS slow and hard ! so during a hackathon, I built gopogh (short for go pretty or go home) that converts




## Github Action example

See [minikube's example](https://github.com/kubernetes/minikube/blob/793eeae748effb7949a2537579b2e4f32a9ab5a8/.github/workflows/master.yml#L162)





### Contribution
Contributons are welcome. 
Run tests:
```
make test
open output.html
```
