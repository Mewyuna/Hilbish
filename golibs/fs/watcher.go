package fs

import (
	"fmt"

	"github.com/rjeczalik/notify"
	rt "github.com/arnodel/golua/runtime"
)

// #type
// Watcher type describes a `fs` library file watcher.
type watcher struct{
	path string
	callback *rt.Closure
	paused bool
	started bool
	ud *rt.UserData
	notifyChan chan notify.EventInfo
	rtm *rt.Runtime
}

func (w *watcher) start() {
	if w.callback == nil || w.started {
		return
	}

	w.started = true
	w.notifyChan = make(chan notify.EventInfo)
	notify.Watch(w.path, w.notifyChan, notify.All)

	go func() {
		for notif := range w.notifyChan {
			ev := notif.Event().String()
			path := notif.Path()

			_, err := rt.Call1(w.rtm.MainThread(), rt.FunctionValue(w.callback), rt.StringValue(ev), rt.StringValue(path))
			if err != nil {
				// TODO: throw error
			}
		}
	}()
}

func (w *watcher) stop() {
	w.started = false
	notify.Stop(w.notifyChan)
}

// #member
// start()
// Start/resume file watching.
func watcherStart(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	pw, err := watcherArg(c, 0)
	if err != nil {
		return nil, err
	}

	pw.start()

	return c.Next(), nil
}

// #member
// stop()
// Stops watching for changes. Effectively ignores changes.
func watcherStop(t *rt.Thread, c *rt.GoCont) (rt.Cont, error) {
	pw, err := watcherArg(c, 0)
	if err != nil {
		return nil, err
	}

	pw.stop()

	return c.Next(), nil
}

func newWatcher(path string, callback *rt.Closure, rtm *rt.Runtime) *watcher {
	pw := &watcher{
		path: path,
		rtm: rtm,
		callback: callback,
	}
	pw.ud = watcherUserData(pw)
	pw.start()

	return pw
}

func watcherArg(c *rt.GoCont, arg int) (*watcher, error) {
	j, ok := valueToWatcher(c.Arg(arg))
	if !ok {
		return nil, fmt.Errorf("#%d must be a watcher", arg + 1)
	}

	return j, nil
}

func valueToWatcher(val rt.Value) (*watcher, bool) {
	u, ok := val.TryUserData()
	if !ok {
		return nil, false
	}

	j, ok := u.Value().(*watcher)
	return j, ok
}

func watcherUserData(j *watcher) *rt.UserData {
	watcherMeta := j.rtm.Registry(watcherMetaKey)
	return rt.NewUserData(j, watcherMeta.AsTable())
}
