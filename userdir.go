package main

import (
	"hilbish/moonlight"
)

// #interface userDir
// user-related directories
// This interface just contains properties to know about certain user directories.
// It is equivalent to XDG on Linux and gets the user's preferred directories
// for configs and data.
// #field config The user's config directory
// #field data The user's directory for program data
func userDirLoader() *moonlight.Table {
	mod := moonlight.NewTable()

	mod.SetField("config", moonlight.StringValue(confDir))
	mod.SetField("data", moonlight.StringValue(userDataDir))

	return mod
}
