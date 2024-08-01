package main

import (
	"hilbish/moonlight"
	//"hilbish/util"

	"github.com/blackfireio/osinfo"
)

// #interface os
// operating system info
// Provides simple text information properties about the current operating system.
// This mainly includes the name and version.
// #field family Family name of the current OS
// #field name Pretty name of the current OS
// #field version Version of the current OS
func hshosLoader() *moonlight.Table {
	info, _ := osinfo.GetOSInfo()
	mod := moonlight.NewTable()

	mod.SetField("family", moonlight.StringValue(info.Family))
	mod.SetField("name", moonlight.StringValue(info.Name))
	mod.SetField("version", moonlight.StringValue(info.Version))

	return mod
}
