#!/bin/sh

## TODO: make a for loop and geneate html for all the .txt files

go tool test2json -t < /data/testout.txt > /data/testout.json 

gopogh -in /data/testout.json -out /data/testout.html "$NAME" -repo "$REPO"  -details "$DETAILS"