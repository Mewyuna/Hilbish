package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"hilbish/moonlight"
	"hilbish/util"

	rt "github.com/arnodel/golua/runtime"
)

var jobs *jobHandler
var jobMetaKey = rt.StringValue("hshjob")

// #type
// #interface jobs
// #property cmd The user entered command string for the job.
// #property running Whether the job is running or not.
// #property id The ID of the job in the job table
// #property pid The Process ID
// #property exitCode The last exit code of the job.
// #property stdout The standard output of the job. This just means the normal logs of the process.
// #property stderr The standard error stream of the process. This (usually) includes error messages of the job.
// The Job type describes a Hilbish job.
type job struct {
	cmd string
	running bool
	id int
	pid int
	exitCode int
	once bool
	args []string
	// save path for a few reasons, one being security (lmao) while the other
	// would just be so itll be the same binary command always (path changes)
	path string
	handle *exec.Cmd
	cmdout io.Writer
	cmderr io.Writer
	stdout *bytes.Buffer
	stderr *bytes.Buffer
	ud *rt.UserData
}

func (j *job) start() error {
	if j.handle == nil || j.once {
		// cmd cant be reused so make a new one
		cmd := exec.Cmd{
			Path: j.path,
			Args: j.args,
		}
		j.setHandle(&cmd)
	}
	// bgProcAttr is defined in execfile_<os>.go, it holds a procattr struct
	// in a simple explanation, it makes signals from hilbish (sigint)
	// not go to it (child process)
	j.handle.SysProcAttr = bgProcAttr
	// reset output buffers
	j.stdout.Reset()
	j.stderr.Reset()
	// make cmd write to both standard output and output buffers for lua access
	j.handle.Stdout = io.MultiWriter(j.cmdout, j.stdout)
	j.handle.Stderr = io.MultiWriter(j.cmderr, j.stderr)

	if !j.once {
		j.once = true
	}

	err := j.handle.Start()
	proc := j.getProc()

	j.pid = proc.Pid
	j.running = true

	hooks.Emit("job.start", rt.UserDataValue(j.ud))

	return err
}

func (j *job) stop() {
	// finish will be called in exec handle
	proc := j.getProc()
	if proc != nil {
		proc.Kill()
	}
}

func (j *job) finish() {
	j.running = false
	hooks.Emit("job.done", rt.UserDataValue(j.ud))
}

func (j *job) wait() {
	j.handle.Wait()
}

func (j *job) setHandle(handle *exec.Cmd) {
	j.handle = handle
	j.args = handle.Args
	j.path = handle.Path
	if handle.Stdout != nil {
		j.cmdout = handle.Stdout
	}
	if handle.Stderr != nil {
		j.cmderr = handle.Stderr
	}
}

func (j *job) getProc() *os.Process {
	handle := j.handle
	if handle != nil {
		return handle.Process
	}

	return nil
}

// #interface jobs
// #member
// start()
// Starts running the job.
func luaStartJob(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	j, err := jobArg(c, 0)
	if err != nil {
		return nil, err
	}

	if !j.running {
		err := j.start()
		exit := handleExecErr(err)
		j.exitCode = int(exit)
		j.finish()
	}

	return c.Next(), nil
}

// #interface jobs
// #member
// stop()
// Stops the job from running.
func luaStopJob(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	j, err := jobArg(c, 0)
	if err != nil {
		return nil, err
	}

	if j.running {
		j.stop()
		j.finish()
	}

	return c.Next(), nil
}

// #interface jobs
// #member
// foreground()
// Puts a job in the foreground. This will cause it to run like it was
// executed normally and wait for it to complete.
func luaForegroundJob(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	j, err := jobArg(c, 0)
	if err != nil {
		return nil, err
	}

	if !j.running {
		return nil, errors.New("job not running")
	}

	// lua code can run in other threads and goroutines, so this exists
	jobs.foreground = true
	// this is kinda funny
	// background continues the process incase it got suspended
	err = j.background()
	if err != nil {
		return nil, err
	}

	err = j.foreground()
	if err != nil {
		return nil, err
	}
	jobs.foreground = false

	return c.Next(), nil
}

// #interface jobs
// #member
// background()
// Puts a job in the background. This acts the same as initially running a job.
func luaBackgroundJob(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}

	j, err := jobArg(c, 0)
	if err != nil {
		return nil, err
	}

	if !j.running {
		return nil, errors.New("job not running")
	}

	err = j.background()
	if err != nil {
		return nil, err
	}

	return c.Next(), nil
}

type jobHandler struct {
	jobs map[int]*job
	latestID int
	foreground bool // if job currently in the foreground
	mu *sync.RWMutex
}

func newJobHandler() *jobHandler {
	return &jobHandler{
		jobs: make(map[int]*job),
		latestID: 0,
		mu: &sync.RWMutex{},
	}
}

func (j *jobHandler) add(cmd string, args []string, path string) *job {
	j.mu.Lock()
	defer j.mu.Unlock()

	j.latestID++
	jb := &job{
		cmd: cmd,
		running: false,
		id: j.latestID,
		args: args,
		path: path,
		cmdout: os.Stdout,
		cmderr: os.Stderr,
		stdout: &bytes.Buffer{},
		stderr: &bytes.Buffer{},
	}
	//jb.ud = jobUserData(jb)

	j.jobs[j.latestID] = jb
	hooks.Emit("job.add", rt.UserDataValue(jb.ud))

	return jb
}

func (j *jobHandler) getLatest() *job {
	j.mu.RLock()
	defer j.mu.RUnlock()

	return j.jobs[j.latestID]
}

func (j *jobHandler) disown(id int) error {
	j.mu.RLock()
	if j.jobs[id] == nil {
		return errors.New("job doesnt exist")
	}
	j.mu.RUnlock()

	j.mu.Lock()
	delete(j.jobs, id)
	j.mu.Unlock()

	return nil
}

func (j *jobHandler) stopAll() {
	j.mu.RLock()
	defer j.mu.RUnlock()

	for _, jb := range j.jobs {
		// on exit, unix shell should send sighup to all jobs
		if jb.running {
			proc := jb.getProc()
			proc.Signal(syscall.SIGHUP)
			jb.wait() // waits for program to exit due to sighup
		}
	}
}

// #interface jobs
// background job management
/*
Manage interactive jobs in Hilbish via Lua.

Jobs are the name of background tasks/commands. A job can be started via
interactive usage or with the functions defined below for use in external runners. */
func (j *jobHandler) loader() *moonlight.Table {
	/*
	jobMethods := rt.NewTable()
	jFuncs := map[string]util.LuaExport{
		"stop": {luaStopJob, 1, false},
		"start": {luaStartJob, 1, false},
		"foreground": {luaForegroundJob, 1, false},
		"background": {luaBackgroundJob, 1, false},
	}
	util.SetExports(l, jobMethods, jFuncs)
	*/

/*
	jobMeta := rt.NewTable()
	jobIndex := func(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
		j, _ := jobArg(c, 0)

		arg := c.Arg(1)
		val := jobMethods.Get(arg)

		if val != rt.NilValue {
			return c.PushingNext1(t.Runtime, val), nil
		}

		keyStr, _ := arg.TryString()

		switch keyStr {
			case "cmd": val = rt.StringValue(j.cmd)
			case "running": val = rt.BoolValue(j.running)
			case "id": val = rt.IntValue(int64(j.id))
			case "pid": val = rt.IntValue(int64(j.pid))
			case "exitCode": val = rt.IntValue(int64(j.exitCode))
			case "stdout": val = rt.StringValue(string(j.stdout.Bytes()))
			case "stderr": val = rt.StringValue(string(j.stderr.Bytes()))
		}

		return c.PushingNext1(t.Runtime, val), nil
	}

	jobMeta.Set(rt.StringValue("__index"), rt.FunctionValue(rt.NewGoFunction(jobIndex, "__index", 2, false)))
	l.SetRegistry(jobMetaKey, rt.TableValue(jobMeta))
*/

	jobFuncs := map[string]moonlight.Export{
		/*
		"all": {j.luaAllJobs, 0, false},
		"last": {j.luaLastJob, 0, false},
		"get": {j.luaGetJob, 1, false},
		"add": {j.luaAddJob, 3, false},
		"disown": {j.luaDisownJob, 1, false},
		*/
	}

	luaJob := moonlight.NewTable()
	l.SetExports(luaJob, jobFuncs)

	return luaJob
}

func jobArg(c *rt.GoCont, arg int) (*job, error) {
	j, ok := valueToJob(c.Arg(arg))
	if !ok {
		return nil, fmt.Errorf("#%d must be a job", arg + 1)
	}

	return j, nil
}

func valueToJob(val rt.Value) (*job, bool) {
	u, ok := val.TryUserData()
	if !ok {
		return nil, false
	}

	j, ok := u.Value().(*job)
	return j, ok
}

/*
func jobUserData(j *job) *rt.UserData {
	jobMeta := l.UnderlyingRuntime().Registry(jobMetaKey)
	return rt.NewUserData(j, jobMeta.AsTable())
}
*/

// #interface jobs
// get(id) -> @Job
// Get a job object via its ID.
// --- @param id number
// --- @returns Job
func (j *jobHandler) luaGetJob(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	jobID, err := c.IntArg(0)
	if err != nil {
		return nil, err
	}

	job := j.jobs[int(jobID)]
	if job == nil {
		return c.Next(), nil
	}

	return c.PushingNext(t.Runtime, rt.UserDataValue(job.ud)), nil
}

// #interface jobs
// add(cmdstr, args, execPath)
// Creates a new job. This function does not run the job. This function is intended to be
// used by runners, but can also be used to create jobs via Lua. Commanders cannot be ran as jobs.
// #param cmdstr string String that a user would write for the job
// #param args table Arguments for the commands. Has to include the name of the command.
// #param execPath string Binary to use to run the command. Needs to be an absolute path.
/*
#example
hilbish.jobs.add('go build', {'go', 'build'}, '/usr/bin/go')
#example
*/
func (j *jobHandler) luaAddJob(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.CheckNArgs(3); err != nil {
		return nil, err
	}
	cmd, err := c.StringArg(0)
	if err != nil {
		return nil, err
	}
	largs, err := c.TableArg(1)
	if err != nil {
		return nil, err
	}
	execPath, err := c.StringArg(2)
	if err != nil {
		return nil, err
	}

	var args []string
	util.ForEach(largs, func(k rt.Value, v rt.Value) {
		if v.Type() == rt.StringType {
			args = append(args, v.AsString())
		}
	})

	jb := j.add(cmd, args, execPath)

	return c.PushingNext1(t.Runtime, rt.UserDataValue(jb.ud)), nil
}

// #interface jobs
// all() -> table[@Job]
// Returns a table of all job objects.
// #returns table[Job]
func (j *jobHandler) luaAllJobs(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	jobTbl := rt.NewTable()
	for id, job := range j.jobs {
		jobTbl.Set(rt.IntValue(int64(id)), rt.UserDataValue(job.ud))
	}

	return c.PushingNext1(t.Runtime, rt.TableValue(jobTbl)), nil
}

// #interface jobs
// disown(id)
// Disowns a job. This simply deletes it from the list of jobs without stopping it.
// #param id number
func (j *jobHandler) luaDisownJob(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	if err := c.Check1Arg(); err != nil {
		return nil, err
	}
	jobID, err := c.IntArg(0)
	if err != nil {
		return nil, err
	}

	err = j.disown(int(jobID))
	if err != nil {
		return nil, err
	}

	return c.Next(), nil
}

// #interface jobs
// last() -> @Job
// Returns the last added job to the table.
// #returns Job
func (j *jobHandler) luaLastJob(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	j.mu.RLock()
	defer j.mu.RUnlock()

	job := j.jobs[j.latestID]
	if job == nil { // incase we dont have any jobs yet
		return c.Next(), nil
	}

	return c.PushingNext1(t.Runtime, rt.UserDataValue(job.ud)), nil
}
