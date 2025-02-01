local fs = require 'fs'

-- explanation: this specific function gives to us info about
-- the currently running source. this includes a path to the
-- source file (info.source)
-- we will use that to automatically load all commands by reading
-- all the files in this dir and just requiring it.
local info = debug.getinfo(1)
local commandDir = fs.dir(info.source)
if commandDir == '.' then return end

local commands = fs.readdir(commandDir)
for _, command in ipairs(commands) do
	local name = command:gsub('%.lua', '') -- chop off extension
	if name ~= 'init' then
		-- skip this file (for obvious reasons)
		require('nature.completions.' .. name)
	end
end

function hilbish.completion.handler(line, pos)
	if type(line) ~= 'string' then error '#1 must be a string' end
	if type(pos) ~= 'number' then error '#2 must be a number' end

	-- trim leading whitespace
	local ctx = line:gsub('^%s*(.-)$', '%1')
	if ctx:len() == 0 then return {}, '' end

	local res = hilbish.aliases.resolve(ctx)
	local resFields = string.split(res, ' ')
	local fields = string.split(ctx, ' ')
	if #fields > 1 and #resFields > 1 then
		fields = resFields
	end
	local query = fields[#fields]

	if #fields == 1 then
		local comps, pfx = hilbish.completion.bins(query, ctx, fields)
		local compGroup = {
			items = comps,
			type = 'grid'
		}

		return {compGroup}, pfx
	else
		local ok, compGroups, pfx = pcall(hilbish.completion.call,
		'command.' .. fields[1], query, ctx, fields)
		if ok then
			return compGroups, pfx
		end

		local comps, pfx = hilbish.completion.files(query, ctx, fields)
		local compGroup = {
			items = comps,
			type = 'grid'
		}

		return {compGroup}, pfx
	end
end
