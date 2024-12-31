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

require 'nature.hilbish'

require 'nature.commands'
require 'nature.completions'
require 'nature.opts'
require 'nature.vim'
require 'nature.runner'
require 'nature.hummingbird'

local shlvl = tonumber(os.getenv 'SHLVL')
if shlvl ~= nil then
	os.setenv('SHLVL', tostring(shlvl + 1))
else
	os.setenv('SHLVL', '0')
end

do
	local virt_G = { }

	setmetatable(_G, {
		__index = function (_, key)
			local got_virt = virt_G[key]
			if got_virt ~= nil then
				return got_virt
			end

			if type(key) == 'string' then
				virt_G[key] = os.getenv(key)
			end
			return virt_G[key]
		end,

		__newindex = function (_, key, value)
			if type(value) == 'string' then
				os.setenv(key, value)
				virt_G[key] = value
			else
				if type(virt_G[key]) == 'string' then
					os.setenv(key, '')
				end
				virt_G[key] = value
			end
		end,
	})
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
