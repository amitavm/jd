package jd

// Structure of a generic POST request; all commands are sent to JDS as POST
// requests.
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
