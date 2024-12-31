local bait = require 'bait'
local commander = require 'commander'
local fs = require 'fs'
local dirs = require 'nature.dirs'

commander.register('cd', function (args, sinks)
	local oldPath = hilbish.cwd()

	if #args > 1 then
		sinks.out:writeln("cd: too many arguments")
		return 1
	end

	local path = args[1] and args[1] or hilbish.home
	if path == '-' then
		path = dirs.old
		sinks.out:writeln(path)
	end

	local ok, err = pcall(function() fs.cd(path) end)
	if not ok then
		sinks.out:writeln(err)
		return 1
	end
	bait.throw('cd', path, oldPath)
	bait.throw('hilbish.cd', fs.abs(path), oldPath)
end)
