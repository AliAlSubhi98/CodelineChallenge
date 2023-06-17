package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"

	"database/sql"

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

	createUserTable := `
		CREATE TABLE user_table (
			id INT PRIMARY KEY,
			name VARCHAR(255),
			last_login_date DATETIME
		)
	`

	createMeasurementResultTable := `
	CREATE TABLE measurement_result_table (
		measurement_result_id INT IDENTITY(1,1) PRIMARY KEY,
		measurement_value VARCHAR(255),
		result_value VARCHAR(255)
	)
`

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

	// Execute the CREATE TABLE statements
	_, err = db.Exec(createUserTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createMeasurementResultTable)
	if err != nil {
		return err
	}

	_, err = db.Exec(createUserActivityTable)
	if err != nil {
		return err
	}

	return nil
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
	err := storeMeasurementResult(result)
	if err != nil {
		log.Println("Error storing measurement result:", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func storeMeasurementResult(result []int) error {
	db, err := getConnection()
	if err != nil {
		return err
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO measurement_result_table (measurement_value, result_value) VALUES (@MeasurementValue, @ResultValue)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, value := range result {
		_, err = stmt.Exec(sql.Named("MeasurementValue", ""), sql.Named("ResultValue", value))
		if err != nil {
			return err
		}
	}

	return nil
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
