package main

import (
	"bytes"
	"iscotDown/encryption"
	"encoding/json"
	"log"
	"net/http"
)

// Currently contacts Lachlan Imel

// When invoked sends a message to the toContact phone number saying the server is down using twilio
func main() {
	encryption.InitConf()
	reporterContact := encryption.GetReporterContact()
	//ownerContact := encryption.GetOwnerContact()
	sendMessage(reporterContact)
	//sendMessage(ownerContact)

}

func sendMessage(destNumber string) {
	phone := encryption.GetPhone()
	message:= "cot is down"
	JSON := BuildJSON(destNumber, phone, message)
	TrySend(JSON)
	
	
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

func TrySend(messageJSON []byte) int {
	// Create an HTTP client
	client := &http.Client{}
	pKey, sKey := encryption.GetAuth()

	// Create an HTTP request
	req, err := http.NewRequest("POST", encryption.GetURL(), bytes.NewBuffer(messageJSON))
	if err != nil {
		log.Fatalf("Error creating HTTP request:", err)
		return -1
	}

	// Set headers for authentication
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(pKey, sKey)

	// Send the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending HTTP request:", err)
		return -1
	}
	defer resp.Body.Close()
	return resp.StatusCode
}

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