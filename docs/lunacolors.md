---
title: Lunacolors
layout: doc
weight: -60
menu: docs
---

Lunacolors is an ANSI color/styling library for Lua. It is included
by default in standard Hilbish distributions to provide easy styling
for things like prompts and text.

For simple usage, a single color or style is enough. For example,
you can just use `lunacolors.blue 'Hello world'` and that'll return
blue text which you can print. This includes styles like bold,
underline, etc.

In other usage, you may want to use a format string instead of having
multiple nested functions for different styles. This is where the format
function comes in. You can used named keywords to style a section of text.

The list of arguments are:  
Colors:
- black
- red
- green
- yellow
- blue
- magenta
- cyan
- white
  
Styles:
- reset
- bold
- dim
- italic
- underline
- invert

For the colors, there are background and bright variants. Background color
variants have a `Bg` suffix, while bright variants use the `bright` prefix.
These can also be combined. Note that appropriate camel casing must be applied. 
For example, bright blue would be written as `brightBlue`, a cyan background as 
`cyanBg`, and combining them would result in `brightBlueBg`.
