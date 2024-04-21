package command

import (
	"crewFinder/db"
	"fmt"
	"log"
	"strings"
)

// Tells the parser what to do based on the status of the user
func ValidateAndParse(input string, phone string, timeStamp int64) string {
	// Convert the text from "+" delimited to " " and change and "%27" to "'"
	input = convertTextMessage(input)
	
	phoneWasDeferred := cleanupDB(phone, timeStamp)

	userStatus := validateUser(phone)

	response := "Error. Invalid user status."

	if phoneWasDeferred {
		userStatus = MEMBER
	}

	switch userStatus {
	case MEMBER:
		if strings.ToLower(input) == "yes" {
			response = "The request you may be responding to is no longer available"
		} else {
			response = parse(input, phone, timeStamp)
		}
	case CONTACTED:
		response = handleContacted(input, phone)
	case INVALID:
		setNameOfMember(phone, input)
		response = "Please confirm: is \"" + input + "\" your name? Type Yes or No."
	case CONFIRMING:
		if strings.ToLower(input) == "yes" {
			setMemberValidity(1, phone)
			// Exactly 160 chars
			response = "Welcome to CoT!\n\nAdd yourself to a role by typing:\nI want to be an MO\n\nAdd yourself to an airport by typing:\nI want to fly from KBLI\n\nType \"help\" for more info"
		} else {
			setNameOfMember(phone, "")
			response = "Please respond with your name to be added to CrewsOutThere."
		}
	case NONMEMBER:
		response = "You are not a member of CrewsOutThere."
	case CLI:
		//For command line
		parse(input, phone, timeStamp)
		//So it doesnt try to send a bad message
		response = "dont run"
	}

	return response
}

// Parses the message and attempts to determine the user's intent.
// Returns a bool that represents if we find a command, along with information needed to fulfil the command.
// Reurns a string that represents the response to send back to the user.
func parse(input string, phone string, timeStamp int64) string {
	// This will be sent back to the user
	response := "Sorry I couldn't understand you. Type \"help\" for help."

	if wantsRegex.MatchString(input) {
		groups := wantsRegex.FindStringSubmatch(input)

		switch groups[2] {
		case "fly":
			if flyNotifRegex.MatchString(input) {
				response = handleFlyNotif(input, phone)
			} else {
				// Shouldn't be possible
				response = "Flight status not changed\n" + FlyUsage()
			}
		case "view":
			if showRegex.MatchString(input) {
				response = handleShow(input, phone)
			} else {
				response = ShowUsage()
			}
		case "invite":
			if inviteRegex.MatchString(input) {
				response = handleInvite(input, phone, timeStamp)
			} else {
				response = "Couldn't invite.\n" + InviteUsage()
			}
		case "be":
			if roleRegex.MatchString(input) {
				response = handleRole(input, phone)
			} else {
				response = "Couldnt set your role.\n" + RoleUsage()
			}
		}

		return response
	}

	if phone == "11" {
		args := strings.Fields(input)
		switch args[0] {
		case "-i":
			response = handleInvite(input, args[1], timeStamp)
		case "-m":
			if args[1] == "-a" {
				//var phone_number string
				query := "SELECT phone_number FROM Members WHERE phone_number != 11"
				selectResult, err := db.DB.Query(query)
				if err != nil {
					log.Fatalf("Impossible select from Members: %s", err)
				}
				var count int
				err2 := db.DB.QueryRow("SELECT COUNT(*) FROM Members WHERE phone_number != 11").Scan(&count)
				if err2 != nil {
					log.Fatalf("Impossible select from Members: %s", err)
				}

				var test = make([]string, count)
				i := 0
				for selectResult.Next() {
					selectResult.Scan(&test[i])
					MessageUser(test[i], args[2], timeStamp, 0)
					i++
				}
			} else {
				MessageUser(args[1], args[2], timeStamp, 0)
			}

		}

		return response
	}

	if helpRegex.MatchString(input) {
		return handleHelp(input)
	}

	if needsRegex.MatchString(input) {
		if requestRegex.MatchString(input) {
			return handleRequest(input, phone, timeStamp)
		}

		// Did not type command correctly
		return "Request not sent.\n" + RequestUsage()
	}

	if parentRegex.MatchString(input) {
		return handleParent(input, phone)
	}

	return response
}

/* Handler functions for the input parser
   Precondition: regex used in the function matches successfully */

// This function determines whether or not the user wants to enable flying notifications
// Returns a string that represents what the system will respond to the user
func handleFlyNotif(input string, phone string) string {
	groups := flyNotifRegex.FindStringSubmatch(input)
	var response string

	enableNotifs := groups[1] == ""
	specifiedAirport := groups[2] != ""
	iataString := strings.ToUpper(groups[3])
	iatas := flyNotifRegexHelper.FindAllStringSubmatch(iataString, -1)

	validIatas := []string{}
	invalidIatas := []string{}

	for _, iata := range iatas {
		// Catch for if users type: "I want to fly kbli"
		if (!specifiedAirport) && (iata[2] != "") {
			response = "Flight status not changed\n" + FlyUsage()
		} else if enableNotifs {

			if specifiedAirport {
				err := addToFlies(iata[2], phone)
				if err != nil {
					response = err.Error()
					invalidIatas = append(invalidIatas, iata[2])
				} else {
					response = "You will now receive notifications for " + iata[2]
					validIatas = append(validIatas, iata[2])
				}

			} else {
				err := updateNotify(phone, 1)
				if err != nil {
					response = err.Error()
					invalidIatas = append(invalidIatas, iata[2])
				} else {
					response = "You will now recieve flight notifications."
					validIatas = append(validIatas, iata[2])
				}
			}
		} else {

			if specifiedAirport {
				err := removeUserAtIATAFromFlies(phone, iata[2])
				if err != nil {
					response = err.Error()
					invalidIatas = append(invalidIatas, iata[2])
				} else {
					response = "You will no longer receive notifications for " + iata[2]
					validIatas = append(validIatas, iata[2])
				}

			} else {
				err := updateNotify(phone, 0)
				if err != nil {
					response = err.Error()
					invalidIatas = append(invalidIatas, iata[2])
				} else {
					response = "You will no longer recieve flight notifications."
					validIatas = append(validIatas, iata[2])
				}
			}
		}
	}

	if len(iatas) != 1 {
		actionVerb := ""
		if enableNotifs {
			actionVerb = "now"
		} else {
			actionVerb = "no longer"
		}
		response = ""

		if len(validIatas) != 0 {
			response += "You will " + actionVerb + " receive notifications for the following airports(s):"
			for _, iata := range validIatas {
				response += " " + iata
			}
		}

		if len(invalidIatas) != 0 {
			if response != "" {
				response += "\n"
			}

			response += "Your notification status could not be changed for the following airports(s):"
			for _, iata := range invalidIatas {
				response += " " + iata
			}
		}
	}

	return response
}

// This function provides a response to send to the user based on some key word
// Returns a string that represents what the system will respond to the user
func handleHelp(input string) string {
	groups := helpRegex.FindStringSubmatch(input)

	// This word is the topic they need help with
	switch strings.ToLower(groups[1]) {
	case "fly":
		return "Enables or disables all flight notifications\n" + FlyUsage()
	case "set role":
		return "Allows you to declare your role\n" + RoleUsage()
	case "view roles":
		return "Displays your roles\n" + ShowRoleUsage()
	case "set airport":
		return "Allows you to set your airports\n" + AirportUsage()
	case "view airports":
		return "Displays your airports\n" + ShowAirportUsage()
	case "invite":
		return "Allows you to invite a number\n" + InviteUsage()
	case "request":
		return "Allows you to request a role for a flight. You will be notified if your request is accepted.\n" + RequestUsage()
	}

	// If word is not in the list (or they just typed "help"), send general help message
	return GeneralUsage()
}

// Lists roles or airports to the user. They can specify if they want to see all roles or airports
// or just their own, with an option to show the details on all of them. Additionally, they can specify
// to just see the details on one given role
func handleShow(input string, phone string) string {
	groups := showRegex.FindStringSubmatch(input)

	table := groups[2]
	showDetails := false

	if groups[2] == "detailed" {
		showDetails = true
		table = groups[3]
	}

	singleItemDetails := table != "roles" && table != "airports"
	filterUser := groups[1] == "my"
	item := groups[1]

	response := ""

	if singleItemDetails {
		response = getDetailsOnRoleOrAirport(item)
	} else {
		switch table {
		case "roles":
			if filterUser {
				response = getEntriesFromWants(phone, showDetails)
			} else {
				response = getAllRoles(showDetails)
			}
		case "airports":
			if filterUser {
				response = getEntriesFromFlies(phone, showDetails)
			} else {
				response = getAllAirports(showDetails)
			}
		}
	}

	return response
}

// This function invites a number to the service
func handleInvite(input string, requestPhone string, timeStamp int64) string {
	groups := inviteRegex.FindStringSubmatch(input)

	phoneNumber := groups[1]
	outcome := inviteUser(requestPhone, phoneNumber, timeStamp)
	if outcome == "" {
		return "Invited User!"
	}
	return outcome
}

// This function allows a user to add or remove a role
func handleRole(input string, phone string) string {
	groups := roleRegex.FindStringSubmatch(input)

	removingRole := groups[1] != ""
	roleString := strings.ToUpper(groups[3])

	if roleString == "" {
		return "Please specify a role."
	}

	roles := roleRegexHelper.FindAllStringSubmatch(roleString, -1)
	validRoles := []string{}
	invalidRoles := []string{}

	response := ""

	for _, role := range roles {
		if !removingRole {
			err := addToWants(role[2], phone)
			if err != nil {
				response = err.Error()
				invalidRoles = append(invalidRoles, role[2])
			} else {
				response = "You are now a " + roleString
				validRoles = append(validRoles, role[2])
			}
		} else {
			err := removeUserAtRoleFromWants(phone, role[2])
			if err != nil {
				response = err.Error()
				invalidRoles = append(invalidRoles, role[2])
			} else {
				response = "You are no longer a " + roleString
				validRoles = append(validRoles, role[2])
			}
		}
	}

	if len(roles) != 1 {
		actionVerb := ""
		if removingRole {
			actionVerb = "removed"
		} else {
			actionVerb = "added"
		}
		response = ""

		if len(validRoles) != 0 {
			response += "You have been " + actionVerb + " to the following role(s):"
			for _, role := range validRoles {
				response += " " + role
			}
		}

		if len(invalidRoles) != 0 {
			if response != "" {
				response += "\n"
			}

			response += "You could not be " + actionVerb + " to the following role(s):"
			for _, role := range invalidRoles {
				response += " " + role
			}
		}
	}

	return response
}

// Initiates a flight request and returns a response to the senders
func handleRequest(input string, phone string, timestamp int64) string {
	groups := requestRegex.FindStringSubmatch(input)

	role := strings.ToUpper(groups[2])
	airport := strings.ToUpper(groups[4])
	message := handleFlyRequest(phone, input, role, airport, timestamp)

	return message
}

// Connects parent and child phone numbers in Parents table
func handleParent(input, phone string) string {
	groups := parentRegex.FindStringSubmatch(input)

	childPhone := "1" + groups[2]

	if groups[1] != "" {
		return removeFromParents(phone, childPhone)
	}
	return addToParents(phone, childPhone)
}

/* End of handler functions */

// Used for testing
func printArray(arr []string) {
	for i, e := range arr {
		fmt.Println(i, "--", e)
	}
}
