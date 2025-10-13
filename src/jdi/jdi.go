package main

import (
	"flag"
	"fmt"
	"strconv"
)

// --- start: globals (constants, variables and data structures.)

// The commands we understand/support.
const (
	CmdNop = "nop" // a no-op; used in place of "no command"
	CmdBwd = "bwd" // jump backward in dirlist
	CmdDir = "dir" // jump to a specified dir
	CmdFwd = "fwd" // jump forward in dirlist
	CmdErr = "err" // send an error message to the client
	CmdIdx = "idx" // jump to a specific index in dirlist
	CmdSyn = "syn" // sync with the dirlist
)

// The "executable" commands.  These can be specified by clients using command
// line flags, and only one of these should be specified on the command line.
var ExecCmds = []string{CmdBwd, CmdDir, CmdFwd, CmdIdx, CmdSyn}

// Process ID of the client process.  Must be specified (using the -c flag).
var clientPID int

// Metadata for commands.
type CmdMetadata struct {
	Flag      string // The command line flag mapped to a command.
	Processor func() // Function to process this command.
}

// CmdData maps command names to their metadata.
//
// The Flag entry for each mapping is set to an empty string here.  They will
// later be auto-populated (where applicable) using the FlagData map below; that
// helps avoid any additional manual work when the set of command line flags is
// modified.
var CmdData = map[string]*CmdMetadata{
	CmdBwd: {"", jumpBwd},
	CmdDir: {"", jumpDir},
	CmdFwd: {"", jumpFwd},
	CmdErr: {"", sendErr},
	CmdIdx: {"", jumpIdx},
	CmdSyn: {"", syncLst},
}

// Metadata for flags.
type FlagMetadata struct {
	IsSet     bool                                      // Is the flag set on the command line?
	Value     any                                       // value set for the flag, or a default
	HelpMsg   string                                    // help message for the flag
	Processor func(fl *flag.Flag, flData *FlagMetadata) // function to process this flag
	Cmd       string                                    // the command mapped to this flag
}

// FlagData maps command line flags to their metadata.
//
// The Value fields are initialized with the default values for the flags.  If a
// non-boolean flag is seen on the command line, its Value field is overridden
// with the value given on the command line.  If a boolean flag is seen on the
// command line, its Value is set to true.
var FlagData = map[string]*FlagMetadata{
	"b": {
		false,
		0,
		"move backward N steps in dirlist",
		processIntFlag,
		CmdBwd,
	},
	"c": {
		false,
		0,
		"PID of the client process",
		processIntFlag,
		CmdNop,
	},
	"d": {
		false,
		"",
		"directory to jump to",
		processStrFlag,
		CmdDir,
	},
	"f": {
		false,
		0,
		"move forward N steps in dirlist",
		processIntFlag,
		CmdFwd,
	},
	"i": {
		false,
		0,
		"jump to dir at index I in dirlist",
		processIntFlag,
		CmdIdx,
	},
	"s": {
		false,
		false,
		"synchronize with the dirlist",
		processBoolFlag,
		CmdSyn,
	},
}

// Xcmd is used to store the command to be executed, as specified on the command
// line. All commands take only one argument (at most).
//
// We normally expect to execute only a single command.  However, in the
// runCommands() function below, we keep executing Xcmd in a for loop till
// IsDone is set to true.  This is done mainly to handle errors: if any error is
// encountered during the execution of a command, we skip setting IsDone to true
// and set Xcmd to CmdErr with an approprirate error message.  This ensures that
// the client receives an appropriate error message indicating the failure.
var Xcmd = struct {
	Name   string // command name
	Arg    any    // argument for the command
	IsDone bool   // whether the command has been executed
}{"", nil, false}

// --- end: globals

// The entry point for JDI.
func main() {
	initialize()

	// Parse and process command line flags.
	registerFlags()
	flag.Parse()
	flag.Visit(processFlag)

	// Now do the main work.
	sanityCheck()
	runCommands()
}

// initialize takes care of all required initializations at the start of a run.
func initialize() {
	for flName, flData := range FlagData {
		if flData.Cmd != CmdNop {
			CmdData[flData.Cmd].Flag = flName
		}
	}
}

// registerFlags registers all our command line flags with the flag package.
func registerFlags() {
	for flName, flData := range FlagData {
		switch flData.Value.(type) {
		case int:
			if flVal, ok := flData.Value.(int); ok {
				flag.IntVar(&flVal, flName, flVal, flData.HelpMsg)
			}
		case bool:
			if flVal, ok := flData.Value.(bool); ok {
				flag.BoolVar(&flVal, flName, flVal, flData.HelpMsg)
			}
		case string:
			if flVal, ok := flData.Value.(string); ok {
				flag.StringVar(&flVal, flName, flVal, flData.HelpMsg)
			}
		default:
			msg := "internal server error: unsupported flag type"
			setCmd(CmdErr, msg)
		}
	}
}

// processFlag processes an individual flag by calling its processor.
func processFlag(fl *flag.Flag) {
	flName := fl.Name
	FlagData[flName].Processor(fl, FlagData[flName])
}

// processIntFlag processes command line flags that take integer arguments.
func processIntFlag(fl *flag.Flag, flData *FlagMetadata) {
	arg := fl.Value.String()
	iarg, err := strconv.Atoi(arg)
	if err != nil {
		msg := fmt.Sprintf("invalid argument to flag '%s': %s\n", fl.Name, arg)
		setCmd(CmdErr, msg)
		return
	}
	finalizeFlag(flData, iarg)
}

// processStrFlag processes command line flags that take string arguments.
func processStrFlag(fl *flag.Flag, flData *FlagMetadata) {
	finalizeFlag(flData, fl.Value.String())
}

// processBoolFlag processes command line flags that take boolean arguments.
func processBoolFlag(fl *flag.Flag, flData *FlagMetadata) {
	finalizeFlag(flData, true)
}

// finalizeFlag does the final processing for a flag. It marks the flag as set,
// saves the argument for the flag, and sets the command to be executed, if any.
func finalizeFlag(flData *FlagMetadata, value any) {
	flData.IsSet = true
	flData.Value = value
	setCmd(flData.Cmd, value)
}

// setCmd sets the command to be executed (for a flag).
func setCmd(cmd string, arg any) {
	// Set only "executable" commands.
	if cmd == CmdNop {
		return
	}

	Xcmd.Name = cmd
	Xcmd.Arg = arg
	Xcmd.IsDone = false
}

// sanityCheck makes sure things look OK.
// It is called before actually running any commands.
func sanityCheck() {
	// Make sure one and only one of the executable commands is requested.
	xCount := 0
	for _, cmd := range ExecCmds {
		if FlagData[CmdData[cmd].Flag].IsSet {
			xCount++
		}
	}

	if xCount == 0 {
		setCmd(CmdErr, "no command specified")
		return
	}

	if xCount > 1 {
		setCmd(CmdErr, "multiple commands specified")
		return
	}

	// Make sure we have the client PID.
	if !FlagData["c"].IsSet {
		setCmd(CmdErr, "client PID not provided")
		return
	} else {
		if pid, ok := FlagData["c"].Value.(int); ok {
			clientPID = pid
		} else {
			setCmd(CmdErr, "invalid client PID provided")
			return
		}
	}
}

// runCommands runs the commands that need to be run.  The commands mostly come
// from the flags specified on the command line, but they can also be generated
// during our own processing.
//
// The most common command of the latter type would be CmdErr when an error is
// encountered.  Errors are also the reason why we run the commands in a for
// loop till the IsDone flag is set to true (see below), even though we expect
// to run only one command most of the time (as set in Xcmd).  When the command
// processors encounter any error, they skip setting IsDone to true and set Xcmd
// to CmdErr with an appropriate error message as its argument.
func runCommands() {
	for !Xcmd.IsDone {
		CmdData[Xcmd.Name].Processor()
	}
}

// --- start: command processors

func jumpBwd() {
	if !IsRightCmd(CmdBwd) {
		return
	}

	if iarg, ok := Xcmd.Arg.(int); ok {
		fmt.Printf("echo \"jump %d steps back in dirlist\"\n", iarg)
		Xcmd.IsDone = true
	} else {
		setCmd(CmdErr, "invalid arg to flag -b")
	}
}

func jumpFwd() {
	if !IsRightCmd(CmdFwd) {
		return
	}

	if iarg, ok := Xcmd.Arg.(int); ok {
		fmt.Printf("echo \"jump %d steps forward in dirlist\"\n", iarg)
		Xcmd.IsDone = true
	} else {
		setCmd(CmdErr, "invalid arg to flag -f")
	}
}

func jumpIdx() {
	if !IsRightCmd(CmdIdx) {
		return
	}

	if iarg, ok := Xcmd.Arg.(int); ok {
		fmt.Printf("echo \"jump to index %d in dirlist\"\n", iarg)
		Xcmd.IsDone = true
	} else {
		setCmd(CmdErr, "invalid arg to flag -i")
	}
}

func jumpDir() {
	if !IsRightCmd(CmdDir) {
		return
	}

	if sarg, ok := Xcmd.Arg.(string); ok {
		fmt.Printf("echo \"jump to dir %s\"\n", sarg)
		Xcmd.IsDone = true
	} else {
		setCmd(CmdErr, "invalid arg to flag -d")
	}
}

func sendErr() {
	if !IsRightCmd(CmdErr) {
		return
	}

	if sarg, ok := Xcmd.Arg.(string); ok {
		fmt.Printf("echo \"jd: error: %s\"\n", sarg)
		Xcmd.IsDone = true
	} else {
		setCmd(CmdErr, fmt.Sprintf("invalid arg for command %s", CmdErr))
	}
}

func syncLst() {
	if !IsRightCmd(CmdSyn) {
		return
	}

	fmt.Printf("echo \"synchronize with dirlist for client %d\"\n", clientPID)
	Xcmd.IsDone = true
}

// --- end: command processors

// IsRightCmd checks if the command set in Xcmd matches a specified command. If
// there is a match, it returns true; otherwise, it sets up an error message to
// be sent back to the client and returns false.
func IsRightCmd(cmd string) bool {
	if Xcmd.Name != cmd {
		msg := fmt.Sprintf("internal jdc error: command '%s' called incorrectly", CmdBwd)
		setCmd(CmdErr, msg)
		return false
	}
	return true
}
