package cli

import (
	"fmt"
	"os"
	"path/filepath"
)

// App is the main structure of a cli application
type App struct {
	// The name of the program. Defaults to path.Base(os.Args[0])
	Name string
	// The version of the program
	Version string
	// Short description of the program.
	Usage string
	// Text to override the USAGE section of help
	UsageText string
	// Long description of the program
	Description string
	// Authors of the program
	Authors string
	// Examples of the program
	Examples string
	// SeeAlso of the program
	SeeAlso string

	// build information, show in --version
	BuildInfo *BuildInfo

	// List of flags to parse
	Flags []*Flag
	// List of commands to execute
	Commands []*Command

	// Hidden --help and --version
	HiddenHelp    bool
	HiddenVersion bool

	AfterExeGlobalCommand bool

	// Display full help
	ShowHelp func(*HelpContext)
	// Display full version
	ShowVersion func(*App)

	// The action to execute when no subcommands are specified
	Action func(*Context)

	// Execute this function if the proper command cannot be found
	OnCommandNotFound func(*Context, string)

	// Handler if panic in app.Action() and command.Action()
	OnActionPanic func(*Context, error)
}

// NewApp creates a new cli Application
func NewApp() *App {
	return &App{
		Name:        filepath.Base(os.Args[0]),
		Usage:       "A new cli application",
		Version:     "0.0.0",
		ShowHelp:    showHelp,
		ShowVersion: showVersion,
	}
}

func (a *App) initialize() {
	// add --help
	if !a.HiddenHelp {
		a.Flags = append(a.Flags, &Flag{
			Name:   "help",
			Usage:  "print this usage",
			IsBool: true,
			Hidden: a.HiddenHelp,
		})
	}

	// add --version
	if !a.HiddenVersion {
		a.Flags = append(a.Flags, &Flag{
			Name:   "version",
			Usage:  "print version information",
			IsBool: true,
			Hidden: a.HiddenVersion,
		})
	}

	// initialize flags
	for _, f := range a.Flags {
		f.initialize()
	}
}

// Run is the entry point to the cli app, parse argument and call Execute() or command.Execute()
func (a *App) Run(arguments []string) {
	a.initialize()

	// parse cli arguments
	cl := &commandline{
		flags:    a.Flags,
		commands: a.Commands,
	}
	err := cl.parse(arguments[1:])

	// build context
	newCtx := &Context{
		name:     a.Name,
		app:      a,
		flags:    a.Flags,
		commands: a.Commands,
		args:     cl.args,
	}

	if err != nil {
		newCtx.ShowError(err)
	}

	// show --help
	if !a.HiddenHelp && newCtx.GetBool("help") {
		newCtx.ShowHelpAndExit(0)
	}
	// show --version
	if !a.HiddenVersion && newCtx.GetBool("version") {
		a.ShowVersion(a)
		os.Exit(0)
	}

	// command not found
	if cl.command == nil && len(a.Commands) > 0 && len(cl.args) > 0 {
		cmd := cl.args[0]
		if a.OnCommandNotFound != nil {
			a.OnCommandNotFound(newCtx, cmd)
		} else {
			newCtx.ShowError(fmt.Errorf("no such command: %s", cmd))
		}
		return
	}

	_globalCommand := func() {
		if a.Action != nil {
			defer newCtx.handlePanic()
			a.Action(newCtx)
		} else if cl.command == nil {
			newCtx.ShowHelpAndExit(0)
		}
	}

	if a.AfterExeGlobalCommand {
		defer _globalCommand()
	} else {
		_globalCommand()
	}

	// run command
	if cl.command != nil {
		cl.command.Run(newCtx)
		return
	}
}
