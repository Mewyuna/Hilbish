<img src="./assets/hilbish-logo-and-text-midnight-edition.png" width=512><br>
<blockquote>
🌓 The Moon-powered shell! A comfy and extensible shell for Lua fans! 🌺 ✨
</blockquote>

<img alt="GitHub commit activity" src="https://img.shields.io/github/commit-activity/m/Rosettea/Hilbish?style=flat-square"><img alt="GitHub commits since latest release (by date)" src="https://img.shields.io/github/commits-since/Rosettea/Hilbish/latest?style=flat-square"><img alt="GitHub contributors" src="https://img.shields.io/github/contributors/Rosettea/Hilbish?style=flat-square"><br>
<a href="https://github.com/Rosettea/Hilbish/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22"><img src="https://img.shields.io/github/issues/Hilbis/Hilbish/help%20wanted?style=flat-square&color=green" alt="help wanted"></a>
<a href="https://github.com/Rosettea/Hilbish/blob/master/LICENSE"><img alt="GitHub license" src="https://img.shields.io/github/license/Rosettea/Hilbish?style=flat-square"></a>
<a href="https://discord.gg/3PDdcQz"><img alt="Discord" src="https://img.shields.io/discord/732357621503229962?color=blue&style=flat-square"></a>
<br>

# Midnight Edition

> [!CAUTION]
> This is a **HEAVILY** WORK IN PROGRESS branch which makes a lot of internal changes.
Functionality **will** be missing if you use this branch,
and you may see crashes too. Tread lightly.

Build instructions and progress on Midnight Edition is tracked in this PR:
[#314](https://github.com/Rosettea/Hilbish/pull/314)

Hilbish: Midinight Edition is a version of Hilbish meant to be compatible with
the original C implementation of Lua by using a Go library binding. The end goal
is to offer Midnight Edition as a separate, "not as supported" build for users
that *really* want to access a certain library made for C Lua or want the
most performance with their Lua code.

**The standard edition, which is all native Go,
will always be more supported than Midnight Edition.**

# Back the original README

Hilbish is an extensible shell designed to be highly customizable.
It is configured in Lua and provides a good range of features.
It aims to be easy to use for anyone but powerful enough for
those who need it.

The motivation for choosing Lua was that its simpler and better to use
than old shell script. It's fine for basic interactive shell uses,
but that's the only place Hilbish has shell script; everything else is Lua
and aims to be infinitely configurable. If something isn't, open an issue!

# Screenshots
<div align="center">
<img src="gallery/tab.png">
<img src="gallery/pillprompt.png">
</div>

# Getting Hilbish
**NOTE:** Hilbish is not guaranteed to work properly on Windows, starting
from the 2.0 version. It will still be able to compile, but functionality
may be lacking. If you want to contribute to make the situation better,
comment on the Windows discussion.

You can check the [install page](https://rosettea.github.io/Hilbish/install/)
on the website for distributed binaries from GitHub or other package repositories.
Otherwise, continue reading for steps on compiling.

## Prerequisites
- [Go 1.22+](https://go.dev)
- [Task](https://taskfile.dev/installation/) (**Go on the hyperlink here to see Task's install method for your OS.**)

## Build
First, clone Hilbish. The recursive is required, as some Lua libraries
are submodules.  
```sh
git clone --recursive https://github.com/Rosettea/Hilbish
cd Hilbish
go get -d ./...
```  

To build, run:
```
go-task
```  

Or, if you want a stable branch, run these commands:
```
git checkout $(git describe --tags `git rev-list --tags --max-count=1`)
go-task build
```  

After you did all that, run `sudo task install` to install Hilbish globally.

# Contributing
Any kind of contributions are welcome! Hilbish is very easy to contribute to.
Read [CONTRIBUTING.md](CONTRIBUTING.md) as a guideline to doing so.

**Thanks to everyone below who's contributed!**  
<a href="https://github.com/Rosettea/Hilbish/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=Rosettea/Hilbish" />
</a>

*Made with [contributors-img](https://contrib.rocks).*

# License
Hilbish is licensed under the [MIT license](LICENSE).  
[Images and assets](assets/) are licensed under CC-BY-SA 4.0
