package command

import (
	"crewFinder/db"
	"database/sql"
	"fmt"
	"log"
)

// Does main work of adding user to flies table
func addToFlies(IATA_Code string, phone string) error {
	if !isIATAInAirports(IATA_Code) {
		return fmt.Errorf("Invalid IATA Code")
	}
	if isMemberAlreadyFlyingAtAirport(phone, IATA_Code) {
		return fmt.Errorf("You are already flying at %s", IATA_Code)
	}

	query := "INSERT INTO Flies (Phone_Number, IATA_Code) VALUES (?, ?)"
	_, err := db.DB.Exec(query, phone, IATA_Code)
	if err != nil {
		log.Fatalf("Error inserting into Flies: %s", err)
	}
	return nil
}

// Does main work of adding user to wants table
func addToWants(roleName string, phone string) error {
	if !isRoleInRoles(roleName) {
		return fmt.Errorf("Invalid role")
	}
	if isMemberAlreadyWantingRole(phone, roleName) {
		return fmt.Errorf("You are already registered as a %s", roleName)
	}

	query := "INSERT INTO Wants (phone_number, Role_Name) VALUES (?, ?)"
	_, err := db.DB.Exec(query, phone, roleName)
	if err != nil {
		log.Fatalf("Error inserting into Wants: %s", err)
	}
	return nil
}

// Removes only the entry where the user is flying at that airport
func removeUserAtIATAFromFlies(phone string, iata string) error {
	if !isIATAInAirports(iata) {
		return fmt.Errorf("The airport %s does not exist", iata)
	}
	if !isMemberAlreadyFlyingAtAirport(phone, iata) {
		return fmt.Errorf("You were not flying at %s", iata)
	}

	query := "DELETE FROM Flies WHERE Phone_Number = ? AND IATA_Code = ?"
	deleteResult, err := db.DB.Exec(query, phone, iata)

	if err != nil {
		log.Fatalf("Impossible delete from Flies: %s", err)
	}

	_, err = deleteResult.RowsAffected()
	if err != nil {
		log.Fatalf("Impossible to know number of rows affected: %s", err)
	}
	return nil
}

// Removes only the entry where the user wants that role
func removeUserAtRoleFromWants(phone string, role string) error {
	if !isRoleInRoles(role) {
		return fmt.Errorf("The role %s does not exist", role)
	}
	if !isMemberAlreadyWantingRole(phone, role) {
		return fmt.Errorf("You were not %s", role)
	}

	query := "DELETE FROM Wants WHERE Phone_Number = ? AND Role_Name = ?"
	deleteResult, err := db.DB.Exec(query, phone, role)

	if err != nil {
		log.Fatalf("Impossible delete from Roles: %s", err)
	}

	_, err = deleteResult.RowsAffected()
	if err != nil {
		log.Fatalf("Impossible to know number of rows affected: %s", err)
	}
	return nil
}

// Update the notify field for a user
func updateNotify(phone string, notify int) error {
	query := "UPDATE Members SET Notify = ? WHERE Phone_Number = ?"
	updateResult, err := db.DB.Exec(query, notify, phone)

	if err != nil {
		log.Fatalf("Impossible to update Members: %s", err)
	}

	_, err = updateResult.RowsAffected()
	if err != nil {
		log.Fatalf("Impossible to know number of rows affected: %s", err)
	}

	return nil
}

// Sets the name of the user to the given name
func setNameOfMember(phone string, name string) {
	query := "UPDATE Members SET name = ? WHERE Phone_Number = ?"
	updateResult, err := db.DB.Exec(query, name, phone)

	if err != nil {
		log.Fatalf("Impossible to update Members: %s", err)
	}

	_, err = updateResult.RowsAffected()
	if err != nil {
		log.Fatalf("Impossible to know number of rows affected: %s", err)
	}
}

// Sets a member to the given state of validity
func setMemberValidity(isValid int, phone string) {
	query := "UPDATE Members SET is_valid = ? WHERE Phone_Number = ?"
	updateResult, err := db.DB.Exec(query, isValid, phone)

	if err != nil {
		log.Fatalf("Impossible to update Members: %s", err)
	}

	_, err = updateResult.RowsAffected()
	if err != nil {
		log.Fatalf("Impossible to know number of rows affected: %s", err)
	}

}

// Takes a user's airports and formats them into a response text
func getEntriesFromFlies(phone string, isDetailed bool) string {
	query := "SELECT iata_code FROM Flies WHERE Phone_Number = ?"
	selectResult, err := db.DB.Query(query, phone)

	if err != nil {
		log.Fatalf("Impossible to select from Flies: %s", err)
	}

	defer selectResult.Close()

	response := ""

	for selectResult.Next() {
		var iata string
		err = selectResult.Scan(&iata)

		if err != nil {
			log.Fatalf("Impossible to get row from selected results: %s", err)
		}

		var comment sql.NullString
		query = "SELECT comment FROM Airports WHERE iata_code = ?"
		err = db.DB.QueryRow(query, iata).Scan(&comment)

		if err != nil {
			log.Fatalf("Error querying database: %s", err)
		}

		if isDetailed {
			response += iata
			if comment.Valid {
				response += ": " + comment.String
			}
			response += "\n"
		} else {
			response += iata + " "
		}
	}

	err = selectResult.Err()
	if err != nil {
		log.Fatalf("Error with select query: %s", err)
	}

	if response == "" {
		response = "You have no airports"
	}

	return response
}

// Takes a user's roles and formats them into a response text
func getEntriesFromWants(phone string, isDetailed bool) string {
	query := "SELECT role_name FROM Wants WHERE Phone_Number = ?"
	selectResult, err := db.DB.Query(query, phone)

	if err != nil {
		log.Fatalf("Impossible to select from Wants: %s", err)
	}

	defer selectResult.Close()

	response := ""

	for selectResult.Next() {
		var role string
		err = selectResult.Scan(&role)

		if err != nil {
			log.Fatalf("Impossible to get row from selected results: %s", err)
		}

		var desc sql.NullString
		query = "SELECT description FROM Roles WHERE role_name = ?"
		err = db.DB.QueryRow(query, role).Scan(&desc)

		if err != nil {
			log.Fatalf("Error querying database: %s", err)
		}

		if isDetailed {
			response += role
			if desc.Valid {
				response += ": " + desc.String
			}
			response += "\n"
		} else {
			response += role + " "
		}
	}

	err = selectResult.Err()
	if err != nil {
		log.Fatalf("Error with select query: %s", err)
	}

	if response == "" {
		response = "You have no roles"
	}

	return response
}

// Takes all airports and formats them into a response text
func getAllAirports(isDetailed bool) string {
	selectResult, err := db.DB.Query("SELECT * FROM Airports")

	if err != nil {
		log.Fatalf("Impossible to select from Airports: %s", err)
	}

	defer selectResult.Close()

	response := ""

	for selectResult.Next() {
		var iata string
		var comment sql.NullString
		err = selectResult.Scan(&iata, &comment)

		if err != nil {
			log.Fatalf("Impossible to get row from selected results: %s", err)
		}

		if isDetailed {
			response += iata
			if comment.Valid {
				response += ": " + comment.String
			}
			response += "\n"
		} else {
			response += iata + " "
		}
	}

	err = selectResult.Err()
	if err != nil {
		log.Fatalf("Error with select query: %s", err)
	}

	return response
}

// Takes all roles and formats them into a response text
func getAllRoles(isDetailed bool) string {
	selectResult, err := db.DB.Query("SELECT * FROM Roles")

	if err != nil {
		log.Fatalf("Impossible to select from Roles: %s", err)
	}

	defer selectResult.Close()

	response := ""

	for selectResult.Next() {
		var role string
		var desc sql.NullString
		err = selectResult.Scan(&role, &desc)

		if err != nil {
			log.Fatalf("Impossible to get row from selected results: %s", err)
		}

		if isDetailed {
			response += role
			if desc.Valid {
				response += ": " + desc.String
			}
			response += "\n"
		} else {
			response += role + " "
		}
	}

	err = selectResult.Err()
	if err != nil {
		log.Fatalf("Error with select query: %s", err)
	}

	return response
}

// Gets the comment or description of the given role or airport
// If somehow there is a role with the same name as an airport, it builds the response
// with string addition so both will be shown and specify which is which
func getDetailsOnRoleOrAirport(item string) string {
	response := ""
	var desc sql.NullString

	query := "SELECT description FROM Roles WHERE role_name = ?"
	err := db.DB.QueryRow(query, item).Scan(&desc)
	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("Error querying database: %s", err)
	}

	if desc.Valid {
		response += item + ": " + desc.String + " (role)\n"
	}

	var comment sql.NullString

	query = "SELECT comment FROM Airports WHERE iata_code = ?"
	err = db.DB.QueryRow(query, item).Scan(&comment)
	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("Error querying database: %s", err)
	}

	if comment.Valid {
		response += item + ": " + comment.String + " (airport)"
	}

	if response == "" {
		response = item + " is not a valid role or airport name"
	}

	return response
}

// Validates the child as the parents invitee and sets the parent-child relationship in the Parents table
func addToParents(parentPhone, childPhone string) string {
	childStatus := validateUser(childPhone)
	if childStatus != MEMBER {
		return "That user is not a member of crewsOutThere. Invite them with \"I want to invite " + childPhone + "\""
	}

	var inviter string
	query := "SELECT added_by FROM Members WHERE phone_number = ?"
	err := db.DB.QueryRow(query, childPhone).Scan(&inviter)
	if err != nil {
		log.Fatalf("Error querying database: %s", err)
	}
	if inviter != parentPhone {
		return "That user was invited by someone else"
	}

	var previousParent string
	query = "SELECT parent_phone FROM Parents WHERE child_phone = ?"
	err = db.DB.QueryRow(query, childPhone).Scan(&previousParent)
	if err == nil {
		if previousParent == parentPhone {
			return "You are already that users parent"
		}
		return "That user already has a parent"
	} else if err != sql.ErrNoRows {
		log.Fatalf("Error querying database")
	}

	query = "INSERT INTO Parents (parent_phone, child_phone) VALUES (?, ?)"
	_, err = db.DB.Exec(query, parentPhone, childPhone)
	if err != nil {
		log.Fatalf("Error inserting into Parents: %s", err)
	}

	return "Added as parent!"
}

// Removes the child-parent relationship if it exists
func removeFromParents(parentPhone, childPhone string) string {
	var previousParent string
	query := "SELECT parent_phone FROM Parents WHERE parent_phone = ? AND child_phone = ?"
	err := db.DB.QueryRow(query, parentPhone, childPhone).Scan(&previousParent)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatalf("Error querying database", err)
		}
		return "You are not registered as that users parent"
	}

	query = "DELETE FROM Parents WHERE parent_phone = ? AND child_phone = ?"
	_, err = db.DB.Exec(query, parentPhone, childPhone)
	if err != nil {
		log.Fatalf("Impossible delete from Parent: %s", err)
	}
	return "You are no longer that users parent"
}

// Invite a user to join crewFinder
func inviteUser(inviter string, invitee string, timeStamp int64) string {
	//Ensure the invitee has not already been invited or added to the system
	if len(invitee) < 11 {
		invitee = "1" + invitee
	}
	if validateUser(invitee) != NONMEMBER {
		return "The person you are trying to invite has already been invited to CrewsOutThere"
	}

	// Get name of the person inviting the user
	inviterName := getNameOfMember(inviter)
	// Build the invite message
	inviteMessage := fmt.Sprintf("%s is inviting you to CrewsOutThere! Please respond with your name to be added. Or respond with \"STOP\" to opt out.", inviterName)
	// Add the unverified user entry into members
	addUnverifiedMember(invitee, inviter, timeStamp)
	// Send the invitee the invite message
	MessageUser(invitee, inviteMessage, timeStamp, 0)
	return ""
}
