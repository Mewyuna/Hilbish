local hilbish = require 'hilbish'
local snail = require 'snail'

hilbish.snail = snail.new()

function hilbish.run(cmd, streams)
	local sinks = {}

	if type(streams) == 'boolean' then
		if not streams then
			sinks = {
				out = hilbish.sink.new(),
				err = hilbish.sink.new(),
				input = io.stdin
			}
		end
	elseif type(streams) == 'table' then
		sinks = streams
	end

	local out = hilbish.snail:run(cmd, {sinks = sinks})
	local returns = {out}

	if type(streams) == 'boolean' and not streams then
		table.insert(returns, sinks.out:readAll())
		table.insert(returns, sinks.err:readAll())
	end

	return table.unpack(returns)
end
