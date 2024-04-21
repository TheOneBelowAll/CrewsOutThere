package command

import (
	"bytes"
	"context"
	"crewFinder/db"
	"crewFinder/encryption"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/sync/semaphore"
)


var Sem *semaphore.Weighted = semaphore.NewWeighted(int64(4096))

// Sends any messages present in the queue table
func SendMessages() {
	var timestamp int64
	var messageJSON []byte
	var priority int64
	var statuscode int
	var queueSize int

	query := "SELECT COUNT(*) FROM Queue"
	err := db.DB.QueryRow(query).Scan(&queueSize)
	if err != nil {
		log.Fatalf("Error querying database: %s", err)
	}

	Sem.Acquire(context.TODO(), int64(4096-queueSize))
	for {
		Sem.Acquire(context.TODO(), 1)
		timestamp, messageJSON, priority = PollMessage()

		statuscode = TrySend(messageJSON)

		query = "DELETE FROM Queue WHERE timestamp = ? AND message = ?"
		_, err = db.DB.Exec(query, timestamp, messageJSON)
		if err != nil {
			log.Fatalf("Error removing from Queue: %s", err)
		}

		// Check the response status
		if statuscode/100 != 2 {
			query = "INSERT INTO Queue (timestamp, message, priority) VALUES (?, ?, ?)"
			_, err = db.DB.Exec(query, timestamp, messageJSON, priority-1)
			if err != nil {
				log.Fatalf("Error inserting into Queue: %s", err)
			}
			Sem.Release(1)
		}
	}
}

func TestSend(messageJSON []byte) int {
	// Create an HTTP client
	client := &http.Client{}

	// Create an HTTP request
	req, err := http.NewRequest("POST", "http://127.0.0.1:3000", bytes.NewBuffer(messageJSON))
	if err != nil {
		log.Fatalf("Error creating HTTP request:", err)
		return -1
	}

	// Set headers for authentication
	req.Header.Set("Content-Type", "application/json")

	// Send the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error sending HTTP request:", err)
		return -1
	}
	defer resp.Body.Close()
	return resp.StatusCode
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

func PollMessage() (int64, []byte, int64) {
	var timestamp int64
	var messageJSON []byte
	var priority int64
	query := "SELECT timestamp, message, priority FROM Queue ORDER BY priority ASC"
	err := db.DB.QueryRow(query).Scan(&timestamp, &messageJSON, &priority)
	if err != nil {
		if err != sql.ErrNoRows {
			log.Fatalf("Error querying database: %s", err)
		}
	}
	return timestamp, messageJSON, priority
}

func ParseJSON(message []byte) MessageData {
	var data MessageData

	err := json.Unmarshal(message, &data)
	if err != nil {
		fmt.Println("Error unmarshalling JSON: ", err)
	}

	return data
}
