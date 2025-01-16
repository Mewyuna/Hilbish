-- Prelude initializes everything else for our shell
local _ = require 'succulent' -- Function additions
local bait = require 'bait'
local fs = require 'fs'

package.path = package.path .. ';' .. hilbish.dataDir .. '/?/init.lua'
.. ';' .. hilbish.dataDir .. '/?/?.lua' .. ";" .. hilbish.dataDir .. '/?.lua'

hilbish.module.paths = '?.so;?/?.so;'
.. hilbish.userDir.data .. 'hilbish/libs/?/?.so'
.. ";" .. hilbish.userDir.data .. 'hilbish/libs/?.so'

table.insert(package.searchers, function(module)
	local path = package.searchpath(module, hilbish.module.paths)
	if not path then return nil end

	-- it didnt work normally, idk
	return function() return hilbish.module.load(path) end, path
end)

require 'nature.commands'
require 'nature.completions'
require 'nature.opts'
require 'nature.vim'
require 'nature.runner'
require 'nature.hummingbird'
require 'nature.env'

local shlvl = tonumber(os.getenv 'SHLVL')
if shlvl ~= nil then
	os.setenv('SHLVL', tostring(shlvl + 1))
else
	os.setenv('SHLVL', '0')
end

do
	local startSearchPath = hilbish.userDir.data .. '/hilbish/start/?/init.lua;'
	.. hilbish.userDir.data .. '/hilbish/start/?.lua'

	local ok, modules = pcall(fs.readdir, hilbish.userDir.data .. '/hilbish/start/')
	if ok then
		for _, module in ipairs(modules) do
			local entry = package.searchpath(module, startSearchPath)
			if entry then
				dofile(entry)
			end
		end
	end

	package.path = package.path .. ';' .. startSearchPath
end

bait.catch('error', function(event, handler, err)
	print(string.format('Encountered an error in %s handler\n%s', event, err:sub(8)))
end)

bait.catch('command.not-found', function(cmd)
	print(string.format('hilbish: %s not found', cmd))
end)

bait.catch('command.not-executable', function(cmd)
	print(string.format('hilbish: %s: not executable', cmd))
end)
