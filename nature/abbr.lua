local bait = require 'bait'
local hilbish = require 'hilbish'
hilbish.abbr = {
	_abbrevs = {}
}

function hilbish.abbr.add(opts)
	hilbish.abbr._abbrevs[opts.abbr] = opts
end

print 'abbr loaded'
hilbish.abbr.add {
	abbr = 'tt',
	expand = 'echo titties'
}

hilbish.abbr.add {
	abbr = 'idk',
	expand = 'i dont know',
	anywhere = true
}

bait.catch('hilbish.rawInput', function(c)
	-- 0x0d == enter
	if c == ' ' or c == string.char(0x0d) then
		-- check if the last "word" was a valid abbreviation
		local line = hilbish.editor.getLine()
		local lineSplits = string.split(line, ' ')
		local thisAbbr = hilbish.abbr._abbrevs[lineSplits[#lineSplits]]

		if thisAbbr and (#lineSplits == 1 or thisAbbr.anywhere == true) then
			hilbish.editor.deleteByAmount(-lineSplits[#lineSplits]:len())
			hilbish.editor.insert(thisAbbr.expand)
		end
	end
end)
