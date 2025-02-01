local function curry(f)
    return function (x) return function (y) return f(x,y) end end
end

local flags = {}
local function flag(f, description)
	flags[f] = {description}
end

local addflag = curry(flag)

addflag '-A' 'Ask for password via askpass or $SUDO_ASKPASS'
addflag '-B' 'Ring the bell as part of the password prompt.'

hilbish.complete('command.sudo', function(query, ctx, fields)
	table.remove(fields, 1)
	local nonflags = table.filter(fields, function(v)
		if v == '' then
			return false
		end
		return v:match '^%-' == nil
	end)

	if #fields == 1 or #nonflags == 0 then
		-- complete commands or sudo flags
		if query:match ('^%-') then
			local compFlags = {}
			for flg, flgstuff in pairs(flags) do
				if flg:match('^' .. query) then
					compFlags[flg] = flgstuff
				end
			end

			local compGroup = {
				items = compFlags,
				type = 'list'
			}

			return {compGroup}, query
		end

		local comps, pfx = hilbish.completion.bins(query, ctx, fields)
		local compGroup = {
			items = comps,
			type = 'grid'
		}

		return {compGroup}, pfx
	end

	-- otherwise, get command flags
	return hilbish.completion.call('command.' .. fields[2], query, ctx, fields)
end)
