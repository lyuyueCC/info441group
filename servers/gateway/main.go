package main

import (
	"log"
	"net/http"
	"os"
)

//main is the main entry point for the server
func main() {
	/* TODO: add code to do the following
	- Read the ADDR environment variable to get the address
	  the server should listen on. If empty, default to ":80"
	- Create a new mux for the web server.
	- Tell the mux to call your handlers.SummaryHandler function
	  when the "/v1/summary" URL path is requested.
	- Start a web server listening on the address you read from
	  the environment variable, using the mux you created as
	  the root handler. Use log.Fatal() to report any errors
	  that occur when trying to start the web server.
	*/

	addr := os.Getenv("ADDR")
	if len(addr) == 0 {
		addr = ":80"
	}
	//create a new mux (router)
	mux := http.NewServeMux()

	//tell it to call the handlers.SummaryHandler function
	//when someone requests the resource path `/v1/summary`
	mux.HandleFunc("/v1/summary", handlers.SummaryHandler)

	//start the web server using the mux as the root handler,
	//and report any errors that occur.
	//the ListenAndServe() function will block so
	//this program will continue to run until killed
	log.Printf("Server is listening on port %s...\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}
