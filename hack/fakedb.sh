#!/bin/bash
echo database record: $1
echo PGHOST=$PGHOST
echo PGUSER=$PGUSER
echo PGPASSWORD=$PGPASSWORD
echo PGSSLMODE=$PGSSLMODE
echo PGDATABASE=$PGDATABASE
# load the fake database into local postgres
psql $PGDATABASE < $1
go run hack/timealign/main.go 
