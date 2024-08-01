module hilbish

go 1.21

toolchain go1.22.2

require (
	github.com/aarzilli/golua v0.0.0-20210507130708-11106aa57765
	github.com/arnodel/golua v0.0.0-20230215163904-e0b5347eaaa1
	github.com/atsushinee/go-markdown-generator v0.0.0-20191121114853-83f9e1f68504
	github.com/blackfireio/osinfo v1.0.5
	github.com/maxlandon/readline v1.0.14
	github.com/pborman/getopt v1.1.0
	github.com/sahilm/fuzzy v0.1.1
	golang.org/x/sys v0.22.0
	golang.org/x/term v0.22.0
	mvdan.cc/sh/v3 v3.8.0
)

require (
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/arnodel/strftime v0.1.6 // indirect
	github.com/evilsocket/islazy v1.11.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/muesli/cancelreader v0.2.2 // indirect
	github.com/olekukonko/ts v0.0.0-20171002115256-78ecb04241c0 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace mvdan.cc/sh/v3 => github.com/Rosettea/sh/v3 v3.4.0-0.dev.0.20240720131751-805c301321fd

replace github.com/maxlandon/readline => ./readline

replace layeh.com/gopher-luar => github.com/layeh/gopher-luar v1.0.10

replace github.com/arnodel/golua => github.com/Rosettea/golua v0.0.0-20240427174124-d239074c1749
