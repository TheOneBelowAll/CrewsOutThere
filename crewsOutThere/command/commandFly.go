package command

import (
	"crewFinder/db"
	"crewFinder/encryption"
	"database/sql"
	"encoding/json"
	"log"
	"strings"
)

/*
If "Yes":
* - send the responders name and phone # back to the original sender (req phone # entry in contacts)
* - add everyone associated with that timestamp's top entry in deferred to contacts (and remove those entries from deferred)
* - send everyone assoc. with that timestamp the message in from their top entry of deferred
* - remove everyone associated with that timestamp from contacts

* If "No":
* - remove them from contacts
* - add their top entry of deferred to contacts (and remove that entry from deferred)
* - send them the message from the top entry of deferred
*/

func handleContacted(input string, contactedPhone string) string {
	requesterPhone, _, timestamp := getItemFromCont(contactedPhone)

	response := ""

	if strings.ToLower(input) == "yes" {
		if GetParent(contactedPhone) == "" {
			removeRequestFromRequesterAtTimestamp(timestamp)
			MessageUser(requesterPhone, buildResponseMessage(contactedPhone), timestamp, 0)
			handleOutgoingContacts(contactedPhone, timestamp)
			response = getNameOfMember(requesterPhone) + " has been notified of your response."
		} else {
			response = "Waiting for parent response"
		}

	} else if strings.ToLower(input) == "no" {
		response = moveSingleEntryFromDefToCont(contactedPhone, timestamp)
	} else {
		// If user did not say yes or no, prompt them again
		response = "Please confirm Yes or No to this request."
	}

	return response
}

// This queries for everyone requested at the given timestamp to tell them that that request has been filled, then
// send them their next request
func handleOutgoingContacts(contactedPhone string, timestamp int64) {
	// First, get all of the people who have been contacted by the original requester as well as the person who
	// accepted the request (in order to send them their next deferred message)

	query := "SELECT * FROM Contacts WHERE timestamp = ?"
	toBeContacted, err := db.DB.Query(query, timestamp)

	if err != nil {
		log.Fatalf("Impossible select from Contacts: %s", err)
	}

	defer toBeContacted.Close()

	// Then, remove the original request from Contacts
	removeRequestFromContactsAtContactedPhone(contactedPhone)

	// Next, for each of the results, move their oldest entry from Deferred into Contacts, then contact them
	for toBeContacted.Next() {
		var rPhone string
		var cPhone string
		err = toBeContacted.Scan(&rPhone, &cPhone, &timestamp)

		if err != nil {
			log.Fatalf("Impossible to get row from selected results: %s", err)
		}

		message := moveSingleEntryFromDefToCont(cPhone, timestamp)
		MessageUser(cPhone, message, timestamp, timestamp)
	}

	err = toBeContacted.Err()
	if err != nil {
		log.Fatalf("Error with select query: %s", err)
	}
}

// Returns the new message to send as found in top entry of def_contacts, or returns that they have no remaining requests
func moveSingleEntryFromDefToCont(contactedPhone string, timestamp int64) string {
	// Get the message from the top Deferred table entry
	newRequesterPhone, _, message, newRequestTimestamp := getTopItemFromDef(contactedPhone)

	parent := GetParent(contactedPhone)
	if parent != "" {
		return ""
	}

	// Remove the old entry from contacts (happens either way) (must happen before the potential new entry is added)
	removeRequestFromContactsAtContactedPhone(contactedPhone)

	child := GetChild(contactedPhone)
	// If user was deferred, add their item to contacts
	if child != "" {
		_, _, _, childRequestTimestamp := getTopItemFromDef(child)
		if childRequestTimestamp == newRequestTimestamp {
			removeRequestFromContactsAtContactedPhone(child)
			if message != "" {
				addItemToContacts(newRequesterPhone, child, newRequestTimestamp)
			}
			removeRequestFromDeferredAtTimestampAndNumber(newRequestTimestamp, child)
			MessageUser(child, message, timestamp, timestamp)
		}
	}

	if message != "" {
		addItemToContacts(newRequesterPhone, contactedPhone, newRequestTimestamp)
	}

	// Remove the new entry from def_cont
	removeRequestFromDeferredAtTimestampAndNumber(newRequestTimestamp, contactedPhone)

	return message
}

type Attributes struct {
	To   string `json:"to"`
	From string `json:"from"`
	Body string `json:"body"`
}
type Data struct {
	Type       string     `json:"type"`
	Attributes Attributes `json:"attributes"`
}
type MessageData struct {
	Data Data `json:"data"`
}

// Sends out the given message to the given phone number
// Splits the message up into groups of 480 chars split at newlines. Required for viewing all roles and airports
// Also will split messages not containing newlines in halves recursively. This will take a while to get back to
// the user, but this should rarely happen
func MessageUser(phone string, message string, timestamp int64, priority int64) {
	if message == "" {
		return
	}
	if len(message) > 480 {
		messageList := strings.Split(message, "\n")
		newMessage := ""

		if len(messageList[0]) > 960 {
			MessageUser(phone, messageList[0][0:len(messageList[0])/2], timestamp, priority)
			MessageUser(phone, messageList[0][len(messageList[0])/2:len(messageList[0])], timestamp, priority)
		} else {
			for i := 0; i < len(messageList); i++ {
				if len(newMessage)+len(messageList[i])+1 > 480 {
					MessageUser(phone, newMessage, timestamp, priority)
					newMessage = ""
				}
				newMessage += messageList[i] + "\n"
			}
			MessageUser(phone, newMessage, timestamp, priority)
		}

	} else {
		addToQueue(phone, message, timestamp, priority)
	}
}

func GetParent(childPhone string) string {
	var parentPhone string
	query := "SELECT parent_phone FROM Parents WHERE child_phone = ?"
	err := db.DB.QueryRow(query, childPhone).Scan(&parentPhone)
	if err != nil {
		if err == sql.ErrNoRows {
			return ""
		}
		log.Fatalf("Error querying database")
	}
	return parentPhone
}

func GetChild(parentPhone string) string {
	var childPhone string
	query := "SELECT child_phone FROM Parents WHERE parent_phone = ?"
	err := db.DB.QueryRow(query, parentPhone).Scan(&childPhone)
	if err != nil {
		if err == sql.ErrNoRows {
			return ""
		}
		log.Fatalf("Error querying database")
	}
	return childPhone
}

// Builds a JSON from a message and adds it to the queue
func addToQueue(to, message string, timestamp, priority int64) {
	messageJSON := BuildJSON(to, encryption.GetPhone(), message)

	query := "INSERT INTO Queue (timestamp, message, priority) VALUES (?, ?, ?)"
	_, err := db.DB.Exec(query, timestamp, messageJSON, priority)
	if err != nil {
		log.Fatalf("Error inserting into Queue: %s", err)
	}
	Sem.Release(1)
}

// Builds JSON message to be sent
func BuildJSON(to, from, message string) []byte {
	messageData := MessageData{
		Data: Data{
			Type: "message",
			Attributes: Attributes{
				To:   to,
				From: from,
				Body: message,
			},
		},
	}

	messageJSON, err := json.Marshal(messageData)
	if err != nil {
		log.Fatalf("Error creating JSON from message data: %s", err)
	}

	return messageJSON
}

// Handles an incoming flight request by messaging relevant users and adding users already in contacted to deferred
func handleFlyRequest(requestPhone string, requestMessage string, role string, airport string, timestamp int64) string {
	// Used to determine if we have found a matching user in which case we add the request to the requester table which also requires a tracking variable as it can happen in two places
	foundMatchingUsers := false

	// Build a properly formatted request message
	requestMessageFull := buildRequestMessage(requestPhone, requestMessage)

	// Add every phone number with matching role and name with number in contacts to be placed in deferred
	query := "SELECT phone_number FROM Flies NATURAL JOIN Wants NATURAL JOIN Members WHERE role_name = ? AND iata_code = ? AND phone_number != ? AND notify = 1 AND phone_number IN (SELECT contacted_phone FROM Contacts)"
	matchingContacted, err := db.DB.Query(query, role, airport, requestPhone)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatalf("Error querying database: %s", err)
		}
	}

	defer matchingContacted.Close()

	// For each user that appears in above query, add them to the deferred table
	for matchingContacted.Next() {
		var contactPhone string
		// Make sure we know we have found matching users
		if !foundMatchingUsers {
			foundMatchingUsers = true
		}
		err = matchingContacted.Scan(&contactPhone)

		if err != nil {
			log.Fatalf("Impossible to get row from selected results: %s", err)
		}

		parent := GetParent(contactPhone)
		if parent != "" {
			if !isMemberAlreadyWantingRole(parent, role) || !isMemberAlreadyFlyingAtAirport(parent, airport) {
				addItemToDeferred(requestPhone, parent, requestMessageFull, timestamp)
			}
		}
		addItemToDeferred(requestPhone, contactPhone, requestMessageFull, timestamp)
	}

	err = matchingContacted.Err()
	if err != nil {
		log.Fatalf("Error with select query: %s", err)
	}

	// If found matching users add request to database
	if foundMatchingUsers {
		addItemToRequester(requestPhone, requestMessageFull, timestamp)
	}
	// Message every phone number with matching role and name with number not in contacts
	query = "SELECT phone_number FROM Flies NATURAL JOIN Wants NATURAL JOIN Members WHERE role_name = ? AND iata_code = ? AND phone_number != ? AND notify = 1 AND phone_number NOT IN (SELECT contacted_phone FROM Contacts)"
	matchingCrew, err := db.DB.Query(query, role, airport, requestPhone)
	if err != nil {
		// It's okay if there's no rows here as there could be valid crew members but they are in contacts
		if err != sql.ErrNoRows {
			log.Fatalf("Error querying database: %s", err)
		}
	}

	defer matchingCrew.Close()

	// Send message out to each user and add them to the contacts table
	for matchingCrew.Next() {
		var contactPhone string
		err = matchingCrew.Scan(&contactPhone)

		// If we haven't added request to database as we hadn't found matching users to add to deferred
		if !foundMatchingUsers {
			foundMatchingUsers = true
			addItemToRequester(requestPhone, requestMessageFull, timestamp)
		}

		if err != nil {
			log.Fatalf("Impossible to get row from selected results: %s", err)
		}

		parent := GetParent(contactPhone)
		if parent != "" {
			if !isMemberAlreadyWantingRole(parent, role) || !isMemberAlreadyFlyingAtAirport(parent, airport) {
				if validateUser(parent) == CONTACTED {
					addItemToDeferred(requestPhone, contactPhone, requestMessageFull, timestamp)
					addItemToDeferred(requestPhone, parent, requestMessageFull, timestamp)
				} else {
					addItemToContacts(requestPhone, contactPhone, timestamp)
					MessageUser(contactPhone, requestMessageFull, timestamp, timestamp)
					addItemToContacts(requestPhone, parent, timestamp)
					MessageUser(parent, requestMessageFull, timestamp, timestamp)
				}
			} else {
				if validateUser(parent) == CONTACTED {
					var ptimestamp int64
					query := "SELECT timestamp from Contacts Where contacted_phone = ?"
					err := db.DB.QueryRow(query, parent).Scan(&ptimestamp)
					if err != nil {
						if err == sql.ErrNoRows {
							log.Fatalf("Parent phone number not in contacts")
						}
						log.Fatalf("Error querying database: %s", err)
					}
					if ptimestamp == timestamp {
						addItemToContacts(requestPhone, contactPhone, timestamp)
						MessageUser(contactPhone, requestMessageFull, timestamp, timestamp)
					} else {
						addItemToDeferred(requestPhone, contactPhone, requestMessageFull, timestamp)
					}
				} else {
					addItemToContacts(requestPhone, contactPhone, timestamp)
					MessageUser(contactPhone, requestMessageFull, timestamp, timestamp)
				}
			}
		} else {
			// First, add to contacts table
			addItemToContacts(requestPhone, contactPhone, timestamp)
			// Then, send message out to user
			MessageUser(contactPhone, requestMessageFull, timestamp, timestamp)
		}
	}

	err = matchingCrew.Err()
	if err != nil {
		log.Fatalf("Error with select query: %s", err)
	}

	if !foundMatchingUsers {
		return "No users were found to be registered under both " + role + " and " + airport + " with notifications on. Your request could not be created."
	}

	return "Request created"
}

// Takes an input message from a user and formats it to be sent out as the request message to other users
func buildRequestMessage(requestPhone string, message string) string {
	var requesterName string
	query := "SELECT Name from Members Where phone_number = ?"
	err := db.DB.QueryRow(query, requestPhone).Scan(&requesterName)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatalf("Requester phone number not in members")
		}
		log.Fatalf("Error querying database: %s", err)
	}
	// TODO Check if user message ends in punctuaotion so if not append a period to validate capital T in type
	requestMessage := requesterName + " is building a crew: " + message + " type Yes or No or ignore."
	return requestMessage
}

// Builds a simple response message based on a contacted user
func buildResponseMessage(contactedPhone string) string {
	name := getNameOfMember(contactedPhone)
	message := name + " (" + contactedPhone + ") has agreed to your request."

	return message
}

// Adds the given information as a new entry in the Contacts table
func addItemToContacts(requesterPhone string, contactedPhone string, timestamp int64) {
	query := "INSERT INTO Contacts (requester_phone, contacted_phone, timestamp) VALUES (?, ?, ?)"
	_, err := db.DB.Exec(query, requesterPhone, contactedPhone, timestamp)
	if err != nil {
		log.Fatalf("Error inserting into Contacts: %s", err)
	}
}

// Adds the given information as a new entry in the requester table
func addItemToRequester(requesterPhone string, request_message string, timestamp int64) {
	query := "INSERT INTO Requester (phone_number, request_message, timestamp) VALUES (?, ?, ?)"
	_, err := db.DB.Exec(query, requesterPhone, request_message, timestamp)
	if err != nil {
		log.Fatalf("Error inserting into Requester: %s", err)
	}
}

// Adds the given information as a new entry in the deferred table
func addItemToDeferred(requesterPhone string, contactedPhone string, requestMessage string, timestamp int64) {
	query := "INSERT INTO Deferred (requester_phone, contacted_phone, request_message, timestamp) VALUES (?, ?, ?, ?)"
	_, err := db.DB.Exec(query, requesterPhone, contactedPhone, requestMessage, timestamp)
	if err != nil {
		log.Fatalf("Error inserting into Deferred: %s", err)
	}
}

// Removes a request entry from the Contacts table at the given timestamp
func removeRequestFromContactsAtContactedPhone(contactedPhone string) {
	query := "DELETE FROM Contacts WHERE contacted_phone = ?"
	_, err := db.DB.Exec(query, contactedPhone)
	if err != nil {
		log.Fatalf("Impossible delete from Contacts: %s", err)
	}
}

// Removes a request entry from the Deferred table at the given timestamp
func removeRequestFromDeferredAtTimestampAndNumber(timestamp int64, contactedPhone string) {
	query := "DELETE FROM Deferred WHERE timestamp = ? AND contacted_phone = ?"
	_, err := db.DB.Exec(query, timestamp, contactedPhone)
	if err != nil {
		log.Fatalf("Impossible delete from Deferred: %s", err)
	}
}

// Removes a request entry from the Deferred table at the given timestamp
func removeRequestFromDeferredAtTimestamp(timestamp int64) {
	query := "DELETE FROM Deferred WHERE timestamp = ?"
	_, err := db.DB.Exec(query, timestamp)
	if err != nil {
		log.Fatalf("Impossible delete from Deferred: %s", err)
	}
}

// Removes a request entry from the Requester table at the given timestamp
func removeRequestFromRequesterAtTimestamp(timestamp int64) {
	query := "DELETE FROM Requester WHERE timestamp = ?"
	_, err := db.DB.Exec(query, timestamp)
	if err != nil {
		log.Fatalf("Impossible delete from Deferred: %s", err)
	}
}

// Returns all of the information in Contacts at the given contacted number
// There will only ever be one entry in contacts for a given contacted number
func getItemFromCont(contactedPhone string) (string, string, int64) {
	var rPhone string
	var cPhone string
	var timestamp int64

	query := "SELECT * FROM Contacts WHERE contacted_phone = ?"
	err := db.DB.QueryRow(query, contactedPhone).Scan(&rPhone, &cPhone, &timestamp)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatalf("Error querying database: %s", err)
		}
	}

	return rPhone, cPhone, timestamp
}

// Returns all of the information from the oldest request entry in Deferred for the given phone number
func getTopItemFromDef(contactedPhone string) (string, string, string, int64) {
	var rPhone string
	var cPhone string
	var message string
	var newRequestTimestamp int64

	query := "SELECT * FROM Deferred WHERE contacted_phone = ? ORDER BY timestamp ASC"
	err := db.DB.QueryRow(query, contactedPhone).Scan(&rPhone, &cPhone, &message, &newRequestTimestamp)
	if err != nil {
		if err == sql.ErrNoRows {
			// This will be used to check if user is not deferred
			message = ""
		} else {
			log.Fatalf("Error querying database: %s", err)
		}
	}

	return rPhone, cPhone, message, newRequestTimestamp
}
