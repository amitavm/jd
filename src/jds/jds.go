package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/amitavm/jd/pkg/jd"
)

// --- start: globals (constants, variables and data structures.)

const (
	// Max size (in bytes) for a (POST request) payload.
	maxPayloadSize = 4 * (1 << 10)
)

// The commands supported/understood by JDS.
const (
	CmdJmpDir = "jmpDir" // jump to a specified directory
	CmdJmpFwd = "jmpFwd" // jump forward in the dirlist a specified number of steps
	CmdJmpBwd = "jmpBwd" // jump backward in the dirlist a specified number of steps
	CmdJmpIdx = "jmpIdx" // jump to the dir at a specified index in the dirlist
	CmdLsDirs = "lsDirs" // provide a listing of the dirlist
)

// Map command names to their processors.
var CmdProcessor = map[string]func(w http.ResponseWriter, req *jd.Request){
	CmdJmpDir: jumpToDir,
	CmdJmpFwd: jumpForward,
	CmdJmpBwd: jumpBackward,
	CmdJmpIdx: jumpToIdx,
	CmdLsDirs: encodeDirList,
}

// Representation of a dirlist.
type DirList struct {
	Name         string    `json:"name"`          // Name of this dirlist.  Optional.
	Dirs         []string  `json:"dirs"`          // The actual list of dirs.
	ClientPID    int       `json:"client_pid"`    // PID of the client process (mostly Bash).
	CurIdx       int       `json:"curidx"`        // Index of the current directory.
	PrevIdx      int       `json:"previdx"`       // Index of the previous directory.
	CreatedAt    time.Time `json:"created_at"`    // When was this dirlist created?
	LoadedAt     time.Time `json:"loaded_at"`     // When was this dirlist loaded?
	LastModified time.Time `json:"last_modified"` // When was this dirlist last modified?
}

// Map client PIDs to their DirList's.
var dirListOf = map[int]*DirList{}

// --- end: globals

// The entry point for JDS.
func main() {
	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/cmd", reqHandler)
	fmt.Println("starting server on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

// rootHandler handles calls to the root URL.
// This is mostly used for testing; it doesn't implement any JD functionality.
func rootHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello from JDS!")
}

// reqHandler handles POST requests for all our "custom" commands.
// It sends a JSON response back using w.
func reqHandler(w http.ResponseWriter, r *http.Request) {
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

	// Parse and process the request.
	var req jd.Request
	if err := decoder.Decode(&req); err != nil {
		msg := fmt.Sprintf("failed to deserialize JSON: %v", err)
		log.Println(msg)
		http.Error(w, "invalid JSON payload: "+msg, http.StatusBadRequest)
		return
	}

	processRequest(w, &req)
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

// processRequest processes all incoming requests/commands.
func processRequest(w http.ResponseWriter, req *jd.Request) {
	CmdProcessor[req.Cmd](w, req)
}

func jumpToDir(w http.ResponseWriter, req *jd.Request) {
	dList := getDirList(req.PID)

	// Add the target dir (in req.Arg) to the end of the dirlist.
	// (We *assume* that the target dir given to us is a valid directory.)
	// And update the previous and current indices, in that order.
	dList.Dirs = append(dList.Dirs, req.Arg)
	dList.PrevIdx = dList.CurIdx
	dList.CurIdx = len(dList.Dirs) - 1

	encodeDirList(w, req)
}

// jumpToIdx jumps to a specific index in the dirlist for a given client (PID).
func jumpToIdx(w http.ResponseWriter, req *jd.Request) {
}

// jumpForward jumps a specified number of steps forward in the dirlist for a
// given client (PID).
func jumpForward(w http.ResponseWriter, req *jd.Request) {
}

// jumpBackward jumps a specified number of steps backward in the dirlist for a
// given client (PID).
func jumpBackward(w http.ResponseWriter, req *jd.Request) {
}

// encodeDirList encodes/serializes the current DirList object for a specified
// client (PID) into the response.
func encodeDirList(w http.ResponseWriter, req *jd.Request) {
	dList := getDirList(req.PID)
	setupResponseHeader(w, http.StatusOK)
	if err := json.NewEncoder(w).Encode(dList); err != nil {
		log.Printf("failed to encode JSON: %v\n", err)
		// This error should almost never occur.
		// But in case it ever does, it will be in the server logs.
	}
}

// setupResponseHeader prepares an HTTP response to be sent back to JDI.
func setupResponseHeader(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/json") // We *always* send JSON responses.
	w.WriteHeader(status)                              // Set the status header to the specified value.
}

// getDirList returns the *DirList for a specified client (PID), by looking it
// up in the global map dirListOf.  If there's no entry for PID in the map, it
// creates a new entry for it with an empty dirlist and returns the same.
func getDirList(pid int) *DirList {
	dList, ok := dirListOf[pid]
	if !ok {
		dList = NewDirList(pid, "")
		dirListOf[pid] = dList
	}
	return dList
}

// NewDirList creates and returns a (pointer to a) new, empty DirList.
func NewDirList(pid int, name string) *DirList {
	now := time.Now()
	return &DirList{
		name,
		[]string{},
		pid,
		-1,
		-1,
		now,
		now,
		now,
	}
}
