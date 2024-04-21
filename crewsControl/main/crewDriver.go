package main

// #cgo CFLAGS:
// #define _DEFAULT_SOURCE
// #include <stdlib.h>
// #include <unistd.h>
// import "C"
import (
	"bufio"
	"crewFinder/command"
	"fmt"
	"os"
	"strings"
	"time"

	"crewFinder/db"
	"crewFinder/encryption"
	"crewFinder/httpServ"
	"log"
	"net/http"
)

// Initialize http handler functions, connection to database, pull values from cot.conf, setup the parser, connect to twilio, and begin serving requests
func main() {
	// C.daemon(1, 0)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	encryption.InitConf()
	db.DBAdminConnect()
	fmt.Println("Entering parserSetup")
	command.ParserSetup()
	http.HandleFunc("/", httpServ.ReceiveText)
	http.HandleFunc("/status", httpServ.ReceiveTest)
	go command.SendMessages()
	go http.ListenAndServe(":3001", nil)

	defaultPhone := "11234567890"
	var currentPhone string
	var inText string
	for {
		scanner := bufio.NewScanner(os.Stdin)

		fmt.Printf("\n\nEnter Text (%s): ", defaultPhone)
		if scanner.Scan() {
			inText = scanner.Text()
		}
		// If user input phone number, parse it for use
		inputPhone, request := splitStringByFirstColon(inText)
		if inputPhone != "" {
			// Need to prepend a 1 to non-default phones to pass verification
			currentPhone = "1" + inputPhone
			inText = request
		} else {
			currentPhone = defaultPhone
		}

		if inText != "" {
			message := command.BuildJSON(encryption.GetPhone(), currentPhone, inText)
			command.TestSend(message)

			time.Sleep(10 * time.Millisecond)
		} else {
			defaultPhone = currentPhone
		}
	}
}

// Used to allow us to specify what phone number we want to message as
func splitStringByFirstColon(input string) (string, string) {
	index := strings.Index(input, ":")
	if index == -1 {
		// Return empty strings if colon is not found
		return "", ""
	}

	phone_number := input[:index]
	request := input[index+1:]

	return phone_number, request
}
