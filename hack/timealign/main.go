package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // Blank import used for registering postgres driver as a database driver

	"github.com/medyagh/gopogh/pkg/models"
)

var dbPath = flag.String("db_path", "", "path to postgres db in the form of 'host=DB_HOST user=DB_USER dbname=DB_NAME password=DB_PASS'")

func main() {
	flag.Parse()
	database, err := sqlx.Connect("postgres", *dbPath)
	if err != nil {
		panic(fmt.Sprintf("failed to open database connection: %v", err))
	}
	if err := modifyTimeStamp(database); err != nil {
		panic(err)
	}

}

// modifyTimeStamp modify all the timestamps in db_environment_test and db_test_cases
// to make all the faked data looks new
func modifyTimeStamp(db *sqlx.DB) error {
	latestTimeStamp, err := getLatestTimeInOldData(db)
	if err != nil {
		return fmt.Errorf("failed to read latest timestamp: %v", err)
	}
	now := time.Now()
	diff := now.Sub(latestTimeStamp)
	days := int(diff.Hours() / 24)
	fmt.Printf("Last timestamp %v, current time %v, difference rounded to days: %d", latestTimeStamp, now, days)
	err = addDaysToAllTime(db, days)
	if err != nil {
		return fmt.Errorf("failed to modify timestamps: %v", err)
	}
	return nil
}

// getLatestTimeInOldData find out the last time stamp in db_environment_test and db_test_cases
func getLatestTimeInOldData(db *sqlx.DB) (time.Time, error) {
	var environmentTest []models.DBEnvironmentTest
	var testCase []models.DBTestCase
	// find the newest time in testtime field of table db_environment_test
	err := db.Select(&environmentTest, "SELECT * FROM db_environment_tests ORDER BY testtime DESC LIMIT 1;")
	if err != nil || len(environmentTest) == 0 {
		return time.Now(), fmt.Errorf("failed to execute SQL query: %v", err)
	}
	// find the newest time in testtime field of table db_test_cases
	err = db.Select(&testCase, "SELECT * FROM db_test_cases ORDER BY testtime DESC LIMIT 1;")
	if err != nil || len(testCase) == 0 {
		return time.Now(), fmt.Errorf("failed to execute SQL query : %v", err)
	}
	// return the latest time
	res := environmentTest[0].TestTime
	if testCase[0].TestTime.Before(res) {
		res = testCase[0].TestTime
	}
	return res, nil
}

// addDaysToAllTime add the interval to all timestamps in db_environment_test and db_test_cases
func addDaysToAllTime(db *sqlx.DB, days int) error {
	results, err := db.Exec(fmt.Sprintf("UPDATE db_test_cases SET testtime=testtime+interval '%d day';", days))
	if err != nil {
		return fmt.Errorf("failed to update testtime: %v", err)
	}
	rowsAffected, err := results.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update testtime: %v", err)
	}
	fmt.Printf("%d rows affected in db_test_cases\n", rowsAffected)

	results, err = db.Exec(fmt.Sprintf("UPDATE db_environment_tests SET testtime=testtime+interval '%d day', gopoghtime=gopoghtime+interval '%d day';", days, days))
	if err != nil {
		return fmt.Errorf("failed to update testtime: %v", err)
	}
	rowsAffected, err = results.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to update testtime: %v", err)
	}
	fmt.Printf("%d rows affected in db_environment_tests\n", rowsAffected)
	return nil
}
