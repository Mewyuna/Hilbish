local fs = require 'fs'
local emmyPattern = '^%-%-%- (.+)'
local modpattern = '^%-+ @module (.+)'
local pieces = {}

local files = fs.readdir 'nature'
for _, fname in ipairs(files) do
	local isScript = fname:match'%.lua$'
	if not isScript then goto continue end

	local f = io.open(string.format('nature/%s', fname))
	local header = f:read '*l'
	local mod = header:match(modpattern)
	if not mod then goto continue end

	print(fname, mod)
	pieces[mod] = {}

	local docPiece = {}
	local lines = {}
	local lineno = 0
	for line in f:lines() do
		lineno = lineno + 1
		lines[lineno] = line

		if line == header then goto continue2 end
		if not line:match(emmyPattern) then
			if line:match '^function' then
				local pattern = (string.format('^function %s%%.', mod) .. '(%w+)')
				local funcName = line:match(pattern)
				if not funcName then goto continue2 end

				local dps = {
					description = {},
					params = {}
				}

				local offset = 1
				while true do
					local prev = lines[lineno - offset]

					local docline = prev:match '^%-+ (.+)'
					if docline then
						local emmy = docline:match '@(%w+)'
						local cut = 0

						if emmy then cut = emmy:len() + 3 end
						local emmythings = string.split(docline:sub(cut), ' ')

						if emmy then
							if emmy == 'param' then
								table.insert(dps.params, 1, {
									name = emmythings[1],
									type = emmythings[2]
								})
							end
						else
							table.insert(dps.description, 1, docline)
						end
						offset = offset + 1
					else
						break
					end
				end

				pieces[mod][funcName] = dps
			end
			docPiece = {}
			goto continue2
		end

		table.insert(docPiece, line)
		::continue2::
	end
	::continue::
end

local header = [[---
title: %s %s
description: %s
layout: doc
menu:
  docs:
    parent: "%s"
---

]]

for iface, dps in pairs(pieces) do
	local mod = iface:match '(%w+)%.' or 'nature'
	local docParent = 'Nature'

	path = string.format('docs/%s/%s.md', mod, iface)
	if mod ~= 'nature' then
		docParent = "API"
		path = string.format('docs/api/%s/%s.md', mod, iface)
	end

	fs.mkdir(fs.dir(path), true)

	local exists = pcall(fs.stat, path)
	local newOrNotNature = exists and mod ~= 'nature'

	local f <close> = io.open(path, newOrNotNature and 'r+' or 'w+')
	if not newOrNotNature then
		f:write(string.format(header, 'Module', iface, 'No description.', docParent))
	end
	print(f)

	print('mod and path:', mod, path)

	local tocSearch = false
	local tocPos
	for line in f:lines() do
		if line:match '^## Functions' then
			tocSearch = true
		end
		if tocSearch and line == '' then
			tocSearch = false
			tocPos = f:seek() - 1
		end
	end

	for func, docs in pairs(dps) do
		local sig = string.format('%s.%s(', iface, func)
		local params = ''
		for idx, param in ipairs(docs.params) do
			sig = sig .. param.name:gsub('%?$', '')
			params = params .. param.name:gsub('%?$', '')
			if idx ~= #docs.params then
				sig = sig .. ', '
				params = params .. ', '
			end
		end
		sig = sig .. ')'

		if tocPos then
			f:seek('set', tocPos)
			local contents = f:read '*a'
			f:seek('set', tocPos)
			local tocLine = string.format('|<a href="#%s">%s</a>|%s|\n', func, string.format('%s(%s)', func, params), docs.description[1])
			f:write(tocLine .. contents)
			f:seek 'end'
		end

		f:write(string.format('<hr>\n<div id=\'%s\'>\n', func))
		f:write(string.format([[
<h4 class='heading'>
%s
<a href="#%s" class='heading-link'>
	<i class="fas fa-paperclip"></i>
</a>
</h4>

]], sig, func))

		f:write(table.concat(docs.description, '\n') .. '\n')
		f:write '#### Parameters\n'
		if #docs.params == 0 then
			f:write 'This function has no parameters.  \n'
		end
		for _, param in ipairs(docs.params) do
			f:write(string.format('`%s` **`%s`**\n', param.name:gsub('%?$', ''), param.type))
		end
		--[[
		local params = table.filter(docs, function(t)
			return t:match '^%-%-%- @param'
		end)
		for i, str in ipairs(params) do
			if i ~= 1 then
				f:write ', '
			end
			f:write(str:match '^%-%-%- @param ([%w]+) ')
		end
		f:write(')\n')

		for _, str in ipairs(docs) do
			if not str:match '^%-%-%- @' then
				f:write(str:match '^%-%-%- (.+)' .. '\n')
			end	
		end
		]]--
		f:write('</div>')
		f:write('\n\n')
	end
end
