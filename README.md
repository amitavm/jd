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

### Switching to UDS

I'm switching to using a Unix Domain Socket (UDS) instead of a TCP port---on
Gemini's suggestion---for the client-server comm.  UDS apparently have the
following benefits over TCP ports:

- There's no port-collision problem.  Multiple users on the same system can run
  multiple instances of JDS without worrying about port collisions.
- UDS are apparently more *secure* than TCP ports?  Something to do with the
  fact that they reside in the filesystem and so are governed by the filesytem
  permissions, etc.  I didn't quite understand this very well though.
- And they are also *faster* because they bypass the network stack?  This one is
  easier to understand, but how much faster, really?  Will it make a differene
  for our use case (interactive human use)?  Still, this is definitely a plus.

Anyway, this *seems* like the right thing to do.  And seems to work too!

This move will also require a change on the client side.  Making the change to
our JDI client should not be hard, as that will require using some well
understood and well documented standard library functions, and we own the code.

However, using external HTTP clients for testing---like cURL---is a different
story: there's no standard way of making them use a UDS instead of a TCP port.
For cURL, the command line that seems to work for me is:

```console
$ curl -s --unix-socket ~/.jd/jd.sock http://localhost/cmd -H 'Content-Type: application/json' -d @cmd.json | jq .
```

The important bit is the `--unix-socket` switch in the command line, specifying
the filesystem node being used for the UDS.  Note that we still specify the full
endpoint URL---and not just `/cmd`, for example---but cURL apparently ignores
the `http://localhost` part, and extracts the endpoint URL `/cmd` from it.

Also, note that it's a POST request, but I haven't explicitly used a `-X POST`
switch in the command line.  cURL is able to deduce it automagically because of
the `-d @cmd.json` switch to send the contents of the `cmd.json` file in the
(POST) request body. In fact, some versions of cURL will explicitly *complain*
about a redundant `-X` command line switch if you do mention it!

(The `jq .` command at the end of the pipeline is to pretty-print the JSON
output. I have shown it here for completeness, but it's not relevant to this
discussion.  You can, of course, use any other JSON pretty-printer in its place;
`python -m json.tool` also works well, provided you have a standard Python
installation.)

So cURL was easy to adapt, but that may not be the case with all other HTTP
clients.  Tools like Postman could be more challenging to work with.  And that's
understandable: many of them are built more for a Windows environment, whereas
UDS is more of a Unix thing.

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
