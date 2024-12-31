-- @module dirs
local bait = require 'bait'
local fs = require 'fs'

local dirs = {}

--- Last (current working) directory. Separate from recentDirs mainly for
--- easier use.
dirs.old = ''
--- Table of recent directories. For use, look at public functions.
dirs.recentDirs = {}
--- Size of the recentDirs table.
dirs.recentSize = 10

--- Get (and remove) a `num` of entries from recent directories.
-- @param num number
-- @param remove boolean Whether to remove items
function dirRecents(num, remove)
	num = num or 1
	local entries = {}

	if #dirs.recentDirs ~= 0 then
		for i = 1, num do
			local idx = remove and 1 or i
			if not dirs.recentDirs[idx] then break end
			table.insert(entries, dirs.recentDirs[idx])
			if remove then table.remove(dirs.recentDirs, 1) end
		end
	end

	if #entries == 1 then
		return entries[1]
	end

	return entries
end

--- Look at `num` amount of recent directories, starting from the latest.
-- @param num? number
function dirs.peak(num)
	return dirRecents(num)
end

--- Add `d` to the recent directories list.
function dirs.push(d)
	dirs.recentDirs[dirs.recentSize + 1] = nil
	if dirs.recentDirs[#dirs.recentDirs - 1] ~= d then
		ok, d = pcall(fs.abs, d)
		assert(ok, 'could not turn "' .. d .. '"into an absolute path')

		table.insert(dirs.recentDirs, 1, d)
	end
end

--- Remove the specified amount of dirs from the recent directories list.
-- @param num number
function dirs.pop(num)
	return dirRecents(num, true)
end

--- Get entry from recent directories list based on index.
-- @param idx number
function dirs.recent(idx)
	return dirs.recentDirs[idx]
end

--- Sets the old directory string.
-- @param d string
function dirs.setOld(d)
	ok, d = pcall(fs.abs, d)
	assert(ok, 'could not turn "' .. d .. '"into an absolute path')

	os.setenv('OLDPWD', d)
	dirs.old = d
end

bait.catch('hilbish.cd', function(path, oldPath)
	dirs.setOld(oldPath)
	dirs.push(path)
end)

return dirs
