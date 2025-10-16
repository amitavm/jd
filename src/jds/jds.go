package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

// --- start: globals (constants, variables and data structures.)

const (
	// Max size (in bytes) for a (POST request) payload.
	maxPayloadSize = 4 * (1 << 10)
)

// Structure of a generic POST request; we get all commands as POST requests.
//
// Note that all commands take one argument at most, and Arg is always a string,
// even for commands that take int arguments. This is done for uniformity of the
// JSON structure. The commands taking int arguments expect the string arg to be
// convertible to an int, otherwise they will return http.StatusBadRequest.
type Request struct {
	Name string `json:"name"` // The actual/specific command name.
	Arg  string `json:"arg"`  // Argument for the command, if any.
	PID  int    `json:"pid"`  // PID of the client process.
}

// Structure of a generic response.
type Response struct {
	Status int    `json:"status"` // The HTTP status code, e.g., http.StatusOK.
	Msg    string `json:"msg"`    // Mostly used for error messages.
}

// --- end: globals

// The entry point for JDS.
func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/cmd", cmdHandler)
	fmt.Println("starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// rootHandler handles calls to the root URL.
// This is mostly used for testing; it doesn't implement any JD functionality.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from JDS!")
}

// cmdHandler handles POST requests for all our "custom" commands.
// It sends a JSON response back using w.
func cmdHandler(w http.ResponseWriter, r *http.Request) {
	// We only handle POST requests.
	if r.Method != http.MethodPost {
		http.Error(w, "invalid request method", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	decoder, err := reqDecoder(w, r)
	if err != nil {
		return
	}

	// Parse and process the request/command.
	var cmd Request
	if err := decoder.Decode(&cmd); err != nil {
		log.Printf("failed to deserialize JSON: %v\n", err)
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}

	if err := processCommand(cmd); err != nil {
		log.Printf("failed to process command: %v\n", err)
		http.Error(w, "failed to process command", http.StatusBadRequest)
		return
	}

	// Send the JSON response.
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := Response{
		Status: http.StatusOK,
		Msg:    fmt.Sprintf("command '%s' processed successfully", cmd.Name),
	}
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("failed to encode JSON: %v\n", err)
		// This error should occur *very* rarely, like almost never.  But in
		// case it ever does, it will be in the server logs.
	}
}

// reqDecoder enforces some best practices for POST requests and returns a
// decoder for the request body.
func reqDecoder(w http.ResponseWriter, r *http.Request) (*json.Decoder, error) {
	// Make sure we are sent JSON content.
	if r.Header.Get("Content-Type") != "application/json" {
		msg := "invalid Content-Type header"
		log.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return nil, errors.New(msg)
	}

	// Put a limit on the payload size.
	r.Body = http.MaxBytesReader(w, r.Body, maxPayloadSize)

	// Be strict about the JSON structures we accept.
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	return decoder, nil
}

// processCommand processes a generic command.
// (This is just a stub for now; the actual implementation will come next.)
func processCommand(cmd Request) error {
	log.Printf("Received command '%s' with arg '%s' [pid %d]\n", cmd.Name, cmd.Arg, cmd.PID)
	return nil
}
