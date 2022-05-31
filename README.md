# Gopogh
[![Github All Releases](https://img.shields.io/github/downloads/medyagh/gopogh/total.svg)]()

Converts golang test results from JSON to user-friendly HTML.

Example test logs:
[before](https://storage.googleapis.com/minikube-builds/logs/13641/22745/Docker_Linux.out.txt),
[after](https://storage.googleapis.com/minikube-builds/logs/13641/22745/Docker_Linux.html).


## Features
- Foldable test results
- Open each sub-test result in a new window
- Sort test by passed/failed/skipped
- Sort test by execution duration
- Search in each test result separately
- Summary table
- Generate json summary


## Give it a try
- First install gopogh:

    ```bash
    go install github.com/medyagh/gopogh/cmd/gopogh@latest
    ```

- Run your integraiton test and convert it to json:

    ```bash
    go tool test2json -t < ./your-test-logs.txt > ./your-test-log.json
    ```

- Run gopogh on it:

    ```bash
    gopogh -in ./your-test-log.json -out_html ./your-test-out.html -out_summary ./your-test-summary.json $TEST_NAME $TEST_PR_NUMBER -repo $REPO_NAME -details $COMMIT_SHA
    ```


## History
I lead the minikube team and due to growing number PRs and integration tests on
multiple OS, drivers and container runtimes, each test failure on a PR generated
tens of thousands of lines for raw logs (with system-level post-mortems) that
made reviewing PRs slow and hard! So, during a hackathon, I built gopogh (short
for go pretty or go home) that converts golang test results from JSON to
user-friendly HTML.


## Github Action example

See [minikube's
example](https://github.com/kubernetes/minikube/blob/793eeae748effb7949a2537579b2e4f32a9ab5a8/.github/workflows/master.yml#L162).


## Contribution
Contributions are welcome. Run tests:

```bash
make test
open output.html
```
