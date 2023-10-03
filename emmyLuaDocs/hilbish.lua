--- @meta

local hilbish = {}

--- Inserts text into the line.
function hilbish.editor.insert(text) end

--- This is an alias (ha) for the `hilbish.alias` function.
--- @param alias string
--- @param cmd string
function hilbish.aliases.add(alias, cmd) end

--- This is the same as the `hilbish.runnerMode` function. It takes a callback,
--- which will be used to execute all interactive input.
--- In normal cases, neither callbacks should be overrided by the user,
--- as the higher level functions listed below this will handle it.
--- @param cb function
function hilbish.runner.setMode(cb) end

--- Calls a completer function. This is mainly used to call
--- a command completer, which will have a `name` in the form
--- of `command.name`, example: `command.git`.
--- You can check `doc completions` for info on the `completionGroups` return value.
--- @param name string
--- @param query string
--- @param ctx string
--- @param fields table
function hilbish.completions.call(name, query, ctx, fields) end

--- The handler function is the callback for tab completion in Hilbish.
--- You can check the completions doc for more info.
--- @param line string
--- @param pos string
function hilbish.completions.handler(line, pos) end

--- Sets an alias of `cmd` to `orig`
--- @param cmd string
--- @param orig string
function hilbish.alias(cmd, orig) end

--- Appends `dir` to $PATH
--- @param dir string|table
function hilbish.appendPath(dir) end

--- Registers a completion handler for `scope`.
--- A `scope` is currently only expected to be `command.<cmd>`,
--- replacing <cmd> with the name of the command (for example `command.git`).
--- `cb` must be a function that returns a table of "completion groups."
--- Check `doc completions` for more information.
--- @param scope string
--- @param cb function
function hilbish.complete(scope, cb) end

--- Returns the current directory of the shell
--- @returns string
function hilbish.cwd() end

--- Replaces running hilbish with `cmd`
--- @param cmd string
function hilbish.exec(cmd) end

--- Puts `fn` in a goroutine
--- @param fn function
function hilbish.goro(fn) end

--- Line highlighter handler. This is mainly for syntax highlighting, but in
--- reality could set the input of the prompt to *display* anything. The
--- callback is passed the current line and is expected to return a line that
--- will be used as the input display.
--- Note that to set a highlighter, one has to override this function.
--- Example:
--- ```
--- function hilbish.highlighter(line)
---    return line:gsub('"%w+"', function(c) return lunacolors.green(c) end)
--- end
--- ```
--- This code will highlight all double quoted strings in green.
--- @param line string
function hilbish.highlighter(line) end

--- The command line hint handler. It gets called on every key insert to
--- determine what text to use as an inline hint. It is passed the current
--- line and cursor position. It is expected to return a string which is used
--- as the text for the hint. This is by default a shim. To set hints,
--- override this function with your custom handler.
--- @param line string
--- @param pos number
function hilbish.hinter(line, pos) end

--- Sets the input mode for Hilbish's line reader. Accepts either emacs or vim
--- @param mode string
function hilbish.inputMode(mode) end

--- Runs the `cb` function every `time` milliseconds.
--- This creates a timer that starts immediately.
--- @param cb function
--- @param time number
--- @return Timer
function hilbish.interval(cb, time) end

--- Changes the continued line prompt to `str`
--- @param str string
function hilbish.multiprompt(str) end

--- Prepends `dir` to $PATH
--- @param dir string
function hilbish.prependPath(dir) end

--- Changes the shell prompt to `str`
--- There are a few verbs that can be used in the prompt text.
--- These will be formatted and replaced with the appropriate values.
--- `%d` - Current working directory
--- `%u` - Name of current user
--- `%h` - Hostname of device
--- @param str string
--- @param typ? string Type of prompt, being left or right. Left by default.
function hilbish.prompt(str, typ) end

--- Read input from the user, using Hilbish's line editor/input reader.
--- This is a separate instance from the one Hilbish actually uses.
--- Returns `input`, will be nil if ctrl + d is pressed, or an error occurs (which shouldn't happen)
--- @param prompt? string
--- @returns string|nil
function hilbish.read(prompt) end

--- Runs `cmd` in Hilbish's sh interpreter.
--- If returnOut is true, the outputs of `cmd` will be returned as the 2nd and
--- 3rd values instead of being outputted to the terminal.
--- @param cmd string
--- @param returnOut boolean
--- @returns number, string, string
function hilbish.run(cmd, returnOut) end

--- Sets the execution/runner mode for interactive Hilbish. This determines whether
--- Hilbish wll try to run input as Lua and/or sh or only do one of either.
--- Accepted values for mode are hybrid (the default), hybridRev (sh first then Lua),
--- sh, and lua. It also accepts a function, to which if it is passed one
--- will call it to execute user input instead.
--- @param mode string|function
function hilbish.runnerMode(mode) end

--- Runs the `cb` function after `time` in milliseconds.
--- This creates a timer that starts immediately.
--- @param cb function
--- @param time number
--- @returns Timer
function hilbish.timeout(cb, time) end

--- Checks if `name` is a valid command.
--- Will return the path of the binary, or a basename if it's a commander.
--- @param name string
--- @returns string
function hilbish.which(name) end

--- Puts a job in the background. This acts the same as initially running a job.
function hilbish.jobs:background() end

--- Returns binary/executale completion candidates based on the provided query.
--- @param query string
--- @param ctx string
--- @param fields table
function hilbish.completions.bins(query, ctx, fields) end

--- Returns file completion candidates based on the provided query.
--- @param query string
--- @param ctx string
--- @param fields table
function hilbish.completions.files(query, ctx, fields) end

--- Puts a job in the foreground. This will cause it to run like it was
--- executed normally and wait for it to complete.
function hilbish.jobs:foreground() end

--- Evaluates `cmd` as Lua input. This is the same as using `dofile`
--- or `load`, but is appropriated for the runner interface.
--- @param cmd string
function hilbish.runner.lua(cmd) end

--- Sets/toggles the option of automatically flushing output.
--- A call with no argument will toggle the value.
--- @param auto boolean|nil
function hilbish:autoFlush(auto) end

--- Flush writes all buffered input to the sink.
function hilbish:flush() end

--- Reads input from the sink.
--- @returns string
function hilbish:read() end

--- Writes data to a sink.
function hilbish:write(str) end

--- Writes data to a sink with a newline at the end.
function hilbish:writeln(str) end

--- Starts running the job.
function hilbish.jobs:start() end

--- Stops the job from running.
function hilbish.jobs:stop() end

--- Runs a command in Hilbish's shell script interpreter.
--- This is the equivalent of using `source`.
--- @param cmd string
function hilbish.runner.sh(cmd) end

--- Starts a timer.
function hilbish.timers:start() end

--- Stops a timer.
function hilbish.timers:stop() end

--- Removes an alias.
--- @param name string
function hilbish.aliases.delete(name) end

--- Get a table of all aliases, with string keys as the alias and the value as the command.
--- @returns table<string, string>
function hilbish.aliases.list() end

--- Tries to resolve an alias to its command.
--- @param alias string
--- @returns string
function hilbish.aliases.resolve(alias) end

--- Adds a new job to the job table. Note that this does not immediately run it.
--- @param cmdstr string
--- @param args table
--- @param execPath string
function hilbish.jobs.add(cmdstr, args, execPath) end

--- Returns a table of all job objects.
--- @returns table<Job>
function hilbish.jobs.all() end

--- Disowns a job. This deletes it from the job table.
--- @param id number
function hilbish.jobs.disown(id) end

--- Get a job object via its ID.
--- @param id number
--- @returns Job
function hilbish.jobs.get(id) end

--- Returns the last added job from the table.
--- @returns Job
function hilbish.jobs.last() end

--- Adds a command to the history.
--- @param cmd string
function hilbish.history.add(cmd) end

--- Retrieves all history.
--- @returns table
function hilbish.history.all() end

--- Deletes all commands from the history.
function hilbish.history.clear() end

--- Retrieves a command from the history based on the `idx`.
--- @param idx number
function hilbish.history.get(idx) end

--- Returns the amount of commands in the history.
--- @returns number
function hilbish.history.size() end

--- Creates a timer that runs based on the specified `time` in milliseconds.
--- The `type` can either be `hilbish.timers.INTERVAL` or `hilbish.timers.TIMEOUT`
--- @param type number
--- @param time number
--- @param callback function
function hilbish.timers.create(type, time, callback) end

--- Retrieves a timer via its ID.
--- @param id number
--- @returns Timer
function hilbish.timers.get(id) end

return hilbish
