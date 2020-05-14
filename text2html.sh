#!/bin/sh

go tool test2json -t < /data/testout.txt > /data/testout.json 
# gopogh -in /data/testout.json -out /data/testout.html -name "${JOB_NAME} ${GITHUB_REF}" -repo "${GITHUB_REPOSITORY}"  -details "${GITHUB_SHA}"
gopogh -in /data/testout.json -out /data/testout.html -name "abcd" -repo "cdfdfd"  -details "sddsfds"