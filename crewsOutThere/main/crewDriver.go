package main

// #cgo CFLAGS:
// #define _DEFAULT_SOURCE
// #include <stdlib.h>
// #include <unistd.h>
// import "C"
import (
	"crewFinder/command"
	"fmt"

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
	err := http.ListenAndServe(":3000", nil)
	log.Fatal(err)
}
