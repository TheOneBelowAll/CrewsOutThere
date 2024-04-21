package commands

import (
	"log"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// Entry function to handle any role related command
func HandleDirect(args []string) {
	if len(args) < 2 {
		log.Fatal("Too few args! For help, invoke this with the flags: '-d -h'")
	}
	switch args[1] {
	case "-i":
		if len(args) < 3 {
			log.Fatalf("Too few args!")
		}
		if len(args) > 3 {
			log.Fatalf("Too many args!")
		}
		cli := args[1] + " " + args[2]
		messageJSON := buildJSON(cli)
		post(messageJSON)
	case "-m":
		if len(args) < 4 {
			log.Fatalf("Too few args!")
		}
		if len(args) > 4 {
			log.Fatalf("Too many args!")
		}
		cli := args[1] + " " + args[2] + " " + args[3]
		messageJSON := buildJSON(cli)
		post(messageJSON)
	case "-h":
		fmt.Println("Usage: ./crewCLI -d -[<option>] [arg] \"[description-optional]\"")
		fmt.Println("\t -i [phone_number] \t invite phone number")
		fmt.Println("\t -m [phone_number] [\"message\"] \t message phone number")
		fmt.Println("\t -m -a [\"message\"] \t message all phone numbers")
		fmt.Println("\t -h \t\t\t direct help")
	default:
		log.Fatalf("Invalid input. For help with direct commands run with -d -h")
	}
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

func buildJSON(cli string) []byte {
	messageData := MessageData{
		Data: Data{
			Type: "message",
			Attributes: Attributes{
				To:   "11",
				From: "11",
				Body: cli,
			},
		},
	}

	messageJSON, err := json.Marshal(messageData)
	if err != nil {
		log.Fatalf("Error creating JSON from message data: %s", err)
	}

	return messageJSON
}

func post(messageJSON []byte) {
	//Use 3000 for CoT and 3001 for CC
	url := "http://127.0.0.1:3000"

	// Create an HTTP client
	client := &http.Client{}

	// Create an HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(messageJSON))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return
	}

	// Send the HTTP request
	resp, err := client.Do(req)
	//fmt.Println(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return
	}
	
	defer resp.Body.Close()
} 

