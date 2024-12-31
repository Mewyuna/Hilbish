--- hilbish.runner
local snail = require 'snail'

local currentRunner = 'hybrid'
local runners = {}

-- lsp shut up
hilbish = hilbish

--- Get a runner by name.
--- @param name string
--- @return table
function hilbish.runner.get(name)
	local r = runners[name]

	if not r then
		error(string.format('runner %s does not exist', name))
	end

	return r
end

--- Adds a runner to the table of available runners. If runner is a table,
--- it must have the run function in it.
--- @param name string
--- @param runner function | table
function hilbish.runner.add(name, runner)
	if type(name) ~= 'string' then
		error 'expected runner name to be a table'
	end

	if type(runner) == 'function' then
		runner = {run = runner} -- this probably looks confusing
	end

	if type(runner) ~= 'table' then
		error 'expected runner to be a table or function'
	end

	if runners[name] then
		error(string.format('runner %s already exists', name))
	end

	hilbish.runner.set(name, runner)
end

--- Sets a runner by name. The runner table must have the run function in it.
--- @param name string
--- @param runner table
function hilbish.runner.set(name, runner)
	if not runner.run or type(runner.run) ~= 'function' then
		error 'run function in runner missing'
	end

	runners[name] = runner
end

--- Executes cmd with a runner. If runnerName isn't passed, it uses
--- the user's current runner.
--- @param cmd string
--- @param runnerName string?
--- @return string, number, string
function hilbish.runner.exec(cmd, runnerName)
	if not runnerName then runnerName = currentRunner end

	local r = hilbish.runner.get(runnerName)

	return r.run(cmd)
end

--- Sets the current interactive/command line runner mode.
--- @param name string
function hilbish.runner.setCurrent(name)
	local r = hilbish.runner.get(name)
	currentRunner = name

	hilbish.runner.setMode(r.run)
end

--- Returns the current runner by name.
--- @returns string
function hilbish.runner.getCurrent()
	return currentRunner
end

function hilbish.runner.sh(input)
	return hilbish.snail:run(input)
end

hilbish.runner.add('hybrid', function(input)
	local cmdStr = hilbish.aliases.resolve(input)

	local res = hilbish.runner.lua(cmdStr)
	if not res.err then
		return res
	end

	return hilbish.runner.sh(input)
end)

hilbish.runner.add('hybridRev', function(input)
	local res = hilbish.runner.sh(input)
	if not res.err then
		return res
	end

	local cmdStr = hilbish.aliases.resolve(input)
	return hilbish.runner.lua(cmdStr)
end)

hilbish.runner.add('lua', function(input)
	local cmdStr = hilbish.aliases.resolve(input)
	return hilbish.runner.lua(cmdStr)
end)

hilbish.runner.add('sh', hilbish.runner.sh)
hilbish.runner.setCurrent 'hybrid'
