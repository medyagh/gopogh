# Contribute Guide
# 1. Gopogh-server
# 1.1 Test in a local database
In reality gopogh server uses the CloudSQL, which may cause some problems for developers if they want to test it locally on a PC. Besides lacking actual data can also influence the local development. Here we provide a guide for to generate a fake database locally with the actual data provided.

### 1.1.1 get the dumped database
[Here](https://storage.googleapis.com/minikube-flake-rate/Backup_Cloud_SQL_Export_2023-10-09%20(11%3A18%3A19).sql) you can download the dumped databse

*NB: This file is about 4.6G. If this database record is too big for your local db, you can modify shirnk them with SQL DELETE command by yourself*


### 1.1.2 start a postgres database locally
You can use any method to achieve this. For example, if you prefer docker containers, you can run
```shell
 docker run -e POSTGRES_PASSWORD=123456 -p 5432:5432 -itd  postgres:latest    
```
### 1.1.3 load database record and modify timestamps
First you need to have at least the following environment variables properly set
- PGHOST
- PGUSER
- PGPASSWORD
- PGSSLMODE
- PGDATABASE
If you have not crated a database in gopogh in postgres, you need to create one first
Then run
```shell
make load-fake-db RECORD_PATH=<path to your database record>
```

#### What this make target is doing
It runs the shell script hack/fakedb.sh, which
- load the dumped database sql record file into postgres using `psql`
- run the script in hack/timealign/main.go, to modify the timestamps of all the records 
*NB: The original dumped DB record is too big and it may take a long time to modify all the records. If that time it no acceptable for you, you can shrink the DB record by yourself*

#### Why we need to modify the timestamp
Gopogh server only present test records within 90 days. So if you only use the database record, which ends at 2023/09/07, you will see nothing. So what we do in that golang script is to find out the duration between today and the latest timestamp, and add the duration to all the records (e.g. if today was 1013/10/07 then we would add 30days to all the timestamps in all the records) 

