env = {}

setmetatable(env, {
	__index = function(_, k)
		return os.getenv(k)
	end,
	__newindex = function(_, k, v)
		os.setenv(k, tostring(v))
	end
})
