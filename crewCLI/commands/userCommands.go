package commands

import (
	"context"
	"crewCLI/db"
	"database/sql"
	"log"
	"fmt"
)

func HandleUsers(args []string) {
	if len(args) != 5 {
		log.Fatal("Too few args! Expected 5")
	}
	if len(args[1]) != 11 {
		log.Fatal("Invalid phone number!, 11-digits enforced")
	}
	var table string
	var column string
	switch args[2] {
	case "-a":
		if !isIATAInAirports(args[4]) {
			log.Fatal("Invalid IATA code")
		}
		table = "Flies"
		column = "iata_code"
	case "-r":
		if !isRoleInRoles(args[4]) {
			log.Fatal("Invalid role")
		}
		table = "Wants"
		column = "role_name"
	case "-h":
		fmt.Println("Usage: ./crewCLI -u \"[phone_number]\" [table] [action] [role_name/IATA]]")
		fmt.Println("\t -a \t use for [table] to select the airport table")
		fmt.Println("\t -r \t use for [table] to select the role table")
		fmt.Println("\t -i \t use for [action] to choose to insert into the chosen table")
		fmt.Println("\t -d \t use for [action] to choose to delete from the chosen table")
		return;
	default:
		log.Fatal("Invalid flag! -a for airports, -r for roles")
	}

	var query string
	switch args[3] {
	case "-i":
		query = "INSERT INTO " + table + "(phone_number, " + column + ") VALUES (?, ?)"
	case "-d":
		query = "DELETE FROM " + table + "WHERE phone_number = ? AND " + column + " = ?"
	default:
		log.Fatal("Invalid flag! -i for insert, -d for delete")
	}
	_, err := db.DB.ExecContext(context.Background(), query, args[1], args[4])
	if err != nil {
		log.Fatalf("Error connecting to database: %s", err)
	}

}

// Check to see if a given IATA Code is in the Airports table
func isIATAInAirports(iata_code string) bool {
	var comment sql.NullString

	query := "SELECT * FROM Airports WHERE iata_code = ?"
	err := db.DB.QueryRow(query, iata_code).Scan(&iata_code, &comment)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		log.Fatalf("Error querying database: %s", err)
	}
	return true
}

// Check to see if a give role name is in roles
func isRoleInRoles(roleName string) bool {
	var roleMessage sql.NullString

	query := "SELECT * FROM Roles WHERE role_name = ?"
	err := db.DB.QueryRow(query, roleName).Scan(&roleName, &roleMessage)
	if err != nil {
		if err == sql.ErrNoRows {
			return false
		}
		log.Fatalf("Error querying database: %s", err)
	}
	return true
}
