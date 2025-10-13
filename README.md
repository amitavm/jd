# JD: Jump to Directory

This is JD, a tool to work with command line shells like Bash and make it easier
to work with multiple directories on the command line.  In the following
discussion, we will only mention Bash, although it should work with all
compatible shells like sh, ksh, zsh, etc.

## Design

JD is built in three parts:

- The frontend, the code that runs in Bash.  This is where the `jd` command
  lives, as a Bash function.
- JDI, the JD Interface that, well, interfaces b/w the frontend and JDS.
- JDS, the JD Server.
