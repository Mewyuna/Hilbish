-- @module hilbish.messages
local bait = require 'bait'
local commander = require 'commander'
local lunacolors = require 'lunacolors'

local M = {}
local counter = 0
local unread = 0
M._messages = {}
M.icons = {
	INFO = '',
	SUCCESS = '',
	WARN = '',
	ERROR = ''
}

hilbish.messages = {}

--- Represents a Hilbish message.
--- @class hilbish.message
--- @field icon string Unicode (preferably standard emoji) icon for the message notification.
--- @field title string Title of the message (like an email subject).
--- @field text string Contents of the message.
--- @field channel string Short identifier of the message. `hilbish` and `hilbish.*` is preserved for internal Hilbish messages.
--- @field summary string A short summary of the message.
--- @field read boolean Whether the full message has been read or not.

function expect(tbl, field)
	if not tbl[field] or tbl[field] == '' then
		error(string.format('expected field %s in message'))
	end
end

--- Sends a message.
--- @param message hilbish.message
function hilbish.messages.send(message)
	expect(message, 'text')
	expect(message, 'title')
	counter = counter + 1
	unread = unread + 1
	message.index = counter
	message.read = false

	M._messages[message.index] = message
	bait.throw('hilbish.notification', message)
end

function hilbish.messages.read(idx)
	local msg = M._messages[idx]
	if msg then 
		M._messages[idx].read = true
		unread = unread - 1
	end
end

function hilbish.messages.readAll(idx)
	for _, msg in ipairs(hilbish.messages.all()) do
		hilbish.messages.read(msg.index)
	end
end

function hilbish.messages.unreadCount()
	return unread
end

function hilbish.messages.delete(idx)
	local msg = M._messages[idx]
	if not msg then
		error(string.format('invalid message index %d', idx or -1))
	end

	M._messages[idx] = nil
end

function hilbish.messages.clear()
	for _, msg in ipairs(hilbish.messages.all()) do
		hilbish.messages.delete(msg.index)
	end
end

function hilbish.messages.all()
	return M._messages
end

return M
