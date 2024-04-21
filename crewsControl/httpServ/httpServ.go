package httpServ

import (
	"crewFinder/command"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"
	//"fmt"
)

// Receive an HTTP Post from FlowRoute, pull out relevant fields
func ReceiveText(w http.ResponseWriter, r *http.Request) {
	var d command.MessageData
	var response string
	timeStamp := time.Now().UnixMilli()
	newErr := json.NewDecoder(r.Body).Decode(&d)
	if newErr != nil {
		log.Fatalf("Error decoding")
		return
	}

	message := d.Data.Attributes.Body
	phone := d.Data.Attributes.From

	parent := command.GetParent(phone)
	if parent != "" {
		command.MessageUser(parent, "("+phone+") "+message, timeStamp, 0)
	}

	response = command.ValidateAndParse(message, phone, timeStamp)

	if response != "dont run" {
		command.MessageUser(phone, response, timeStamp, 0)
	}

}

// This is used so we know the server is still running, responds to crontab pings
func ReceiveTest(w http.ResponseWriter, r *http.Request) {
	_, err := io.WriteString(w, "running")
	if err != nil {
		log.Println("Err from ReceiveTest: ", err)
	}
}
