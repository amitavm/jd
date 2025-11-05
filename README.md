# JD: Jump to Directory

This is JD: "Jump to Directory".  It is intended to work with command shells
like Bash and make it easier to work with multiple directories on the command
line.

In the following discussion, we will only mention Bash, although it should work
well with all Bourne-compatible shells like sh, ksh, zsh, etc.

## Overview

You can use JD as a replacement for the shell's `cd` command.  The builtin `cd`
command in command shells like Bash makes it somewhat convenient to work with
two directories at a time, using the `cd -` command that lets you jump back and
forth between any two directories without having to type their (absolute or
relative) pathnames every time.  But if you want to work with more than two
directories, you will need to keep typing their pathnames multiple times on the
command line.

<!-- TODO: Mention pushd/popd. -->

## Design

JD is built in three parts:

- The frontend (FE), the code that runs in Bash.  This is where the user command
  `jd` lives, as a Bash function.
- JDI, the JD Interface that, well, interfaces b/w the FE and JDS (the JD
  Server, described below).  It takes commands from the FE, and translates them
  to appropriate HTTP requests to be sent to JDS.  It similarly translates the
  responses from JDS to commands that Bash can execute.
- JDS, the JD Server.  It is implemented as an HTTP API.  It runs in the
  background as a server, maintains the directory lists for any number of client
  processes (presumably command shells), and also performs all the "business
  logic" required to implement the various features of JD.

### JDS

JDS implements an HTTP based API, for the simple reason that it's easy to do
that in most modern languages.  (We are using Go.)  But please note that it's
only an HTTP API, and not the usual REST kind of API: it doesn't implement the
usual CRUD methods.

By design, JDS needs to support a variety of commands, many more than the number
of HTTP verbs.  We also want to make it possible to add more commands in the
future, if need be.  For these reasons, we have used a simple (and not uncommon)
convention of using a POST endpoint called `cmd` to implement all such "custom"
commands.

The response from JDS is always the current, complete state of the
dirlist---unless some error occurs.

## Control Flow

In this section, we outline some representative use cases and how the control
flows between the three JD parts in each case.

### At Startup

At startup, the FE logs in to JDS (on behalf of its Bash instance) and obtains a
login token.  All subsequent communications between the FE and JDI/JDS include
this login token to uniquely identify the client process (Bash).

### Command `jd DIR`

The following happens when the user types the command `jd DIR` on the command
line, where DIR stands for the (absolute or relative) pathname of a directory.

- The FE (running in Bash) checks to make sure DIR is a valid directory.
  It then replaces DIR with its absolute pathname, if required.
- (This could also be done in JDI, but it's so much easier to do it in Bash.)
- It then invokes JDI using the command line `jdi -d DIR`.
- (Bash spawns a new instance of JDI to process this command line, of course.)
- JDI parses the command line flag (`-d DIR`) and sends the following JSON
  request (sans the comments, of course) to JDS in a POST call:

  ```jsonc
  {
    "cmd": "jumpDir",
    "arg": "DIR"      // the directory to jump to
    "pid": "PID"      // process ID of the client (Bash)
  }
  ```

  We send the PID of the client process (Bash) with each request to JDS because
  JDS maintains directory lists for multiple shells, so it needs to know the
  specific shell instance it needs to work on behalf of.

### Command `jd -l`

This command is meant to print a listing of the current directory list to the
screen.  This is what happens behind the scenes to make this work:

- The FE just relays the command line flag to JDI using `jdi -l`.
- JDI parses the flag and sends the following JSON to JDS:

  ```jsonc
    "cmd": "lsDirs",
    "arg": ""         // no argument required in this case
    "pid": PID        // process ID of the client (Bash)
  ```


## Compatibility with `cd`.

JD, being a replacement for the shell's builtin `cd` command, mimics the
behavior of all valid `cd` commands:

- When the user types just `jd` on the command line, JD behaves as if the user
  had typed `jd $HOME` instead.
- When the user types `cd -`, JD jumps to the "previous" directory the user was
  in.
