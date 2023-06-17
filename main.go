package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"

	_ "github.com/denisenkom/go-mssqldb" // MS SQL Server driver
)

func getConnection() (*sql.DB, error) {
	server := "CODELINE002"
	port := 1433
	database := "CodelineChallenge1"
	user := "sa"
	password := "root"

	connectionString := fmt.Sprintf("server=%s;port=%d;database=%s;user id=%s;password=%s",
		server, port, database, user, password)

	db, err := sql.Open("sqlserver", connectionString)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func createTables() error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if the user_table exists
	if tableExists(db, "user_table") {
		fmt.Println("user_table already exists")
	} else {
		// Create the user_table
		createUserTable := `
		CREATE TABLE user_table (
			id INT PRIMARY KEY,
			name VARCHAR(255),
			last_login_date DATETIME
		)
		`

		_, err := db.Exec(createUserTable)
		if err != nil {
			return err
		}
		fmt.Println("user_table created")
	}

	// Check if the measurement_result_table exists
	if tableExists(db, "measurement_result_table") {
		fmt.Println("measurement_result_table already exists")
	} else {
		// Create the measurement_result_table
		createMeasurementResultTable := `
		CREATE TABLE measurement_result_table (
			measurement_result_id INT IDENTITY(1,1) PRIMARY KEY,
			measurement_value VARCHAR(255),
			result_value VARCHAR(255)
		)
		`

		_, err := db.Exec(createMeasurementResultTable)
		if err != nil {
			return err
		}
		fmt.Println("measurement_result_table created")
	}

	// Check if the user_activity_table exists
	if tableExists(db, "user_activity_table") {
		fmt.Println("user_activity_table already exists")
	} else {
		// Create the user_activity_table
		createUserActivityTable := `
		CREATE TABLE user_activity_table (
			datetime DATETIME,
			user_id INT,
			username VARCHAR(255),
			measurement_value VARCHAR(255),
			measurement_result_id INT,
			FOREIGN KEY (user_id) REFERENCES user_table(id),
			FOREIGN KEY (measurement_result_id) REFERENCES measurement_result_table(measurement_result_id)
		)
		`

		_, err := db.Exec(createUserActivityTable)
		if err != nil {
			return err
		}
		fmt.Println("user_activity_table created")
	}

	return nil
}

func tableExists(db *sql.DB, tableName string) bool {
	query := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = '%s'", tableName)
	var count int
	err := db.QueryRow(query).Scan(&count)
	if err != nil {
		log.Println("Error checking table existence:", err)
		return false
	}
	return count > 0
}

func main() {
	err := createTables()
	if err != nil {
		fmt.Println("Error creating tables:", err)
	} else {
		fmt.Println("Tables created successfully")
	}

	http.HandleFunc("/convert-measurements", convertMeasurementsHandler)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs) // Serve static files
	log.Println("Server started on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func convertMeasurementsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	measurements := r.FormValue("convert-measurements")
	result := convertMeasurements(measurements)
	response := struct {
		Result []int `json:"result"`
	}{
		Result: result,
	}

	// Store the measurement conversion result in the database
	err := storeMeasurementResult(measurements, result)
	if err != nil {
		log.Println("Error storing measurement result:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func storeMeasurementResult(measurementValue string, result []int) error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	// Prepare the SQL statement
	stmt, err := db.Prepare("INSERT INTO measurement_result_table (measurement_value, result_value) VALUES (@MeasurementValue, @ResultValue)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Start a transaction to ensure atomicity
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	// Execute the SQL statement once with all the result values
	_, err = stmt.Exec(sql.Named("MeasurementValue", measurementValue), sql.Named("ResultValue", resultToString(result)))
	if err != nil {
		tx.Rollback() // Rollback the transaction if an error occurs
		return err
	}

	// Commit the transaction if the execution is successful
	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func resultToString(result []int) string {
	str := ""
	for i, value := range result {
		if i > 0 {
			str += ","
		}
		str += fmt.Sprintf("%d", value)
	}
	return str
}

func convertMeasurements(str string) []int {
	collectedValues := make([]int, 0)
	if isValidSeq(str) {
		isNewNumber := true
		isValAfterZ := false
		totalZValues := 0
		roundLength := 0
		roundItr := 0
		roundTotal := 0
		charVal := 0

		for i := 0; i < len(str); i++ {
			char := str[i]
			if char == '_' {
				charVal = 0
				if isValAfterZ {
					charVal += totalZValues
					totalZValues = 0
					isValAfterZ = false
				}
			} else if char == 'z' {
				isValAfterZ = true
				totalZValues += charVal
				continue
			} else {
				charVal = int(char) - 96
				if isValAfterZ {
					charVal += totalZValues
					totalZValues = 0
					isValAfterZ = false
				}
			}

			if !isNewNumber {
				roundTotal += charVal
				roundItr++
			} else {
				roundLength = charVal
				roundTotal = 0
				roundItr = 0
				isNewNumber = false
			}

			if roundItr == roundLength {
				collectedValues = append(collectedValues, roundTotal)
				isNewNumber = true
			}

			if i == len(str)-1 && roundItr != roundLength {
				collectedValues = append(collectedValues, 0)
			}
		}
	}
	return collectedValues
}

func isValidSeq(str string) bool {
	pattern := "^[a-z_]+$"
	match, _ := regexp.MatchString(pattern, str)
	return match
}
