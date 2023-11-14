#!/bin/sh

(
cat <<EOF
/*
Command mox is a modern, secure, full-featured, open source mail server for
low-maintenance self-hosted email.

# Commands

EOF

./mox 2>&1 | sed 's/^\( *\|usage: \)/\t/'

cat <<EOF

Many commands talk to a running mox instance, through the ctl file in the data
directory. Specify the configuration file (that holds the path to the data
directory) through the -config flag or MOXCONF environment variable.

EOF

# setting XDG_CONFIG_HOME ensures "mox localserve" has reasonable default
# values in its help output.
XDG_CONFIG_HOME='$userconfigdir' ./mox helpall 2>&1

cat <<EOF
*/
package main

// NOTE: DO NOT EDIT, this file is generated by gendoc.sh.
EOF
)>doc.go
gofmt -w doc.go

(
cat <<EOF
/*
Package config holds the configuration file definitions for mox.conf (Static)
and domains.conf (Dynamic).

These config files are in "sconf" format.  Summarized: Indent with tabs, "#" as
first non-whitespace character makes the line a comment (you cannot have a line
with both a value and a comment), strings are not quoted/escaped and can never
span multiple lines. See https://pkg.go.dev/github.com/mjl-/sconf for details.

Annotated empty/default configuration files you could use as a starting point
for your mox.conf and domains.conf, as generated by "mox config
describe-static" and "mox config describe-domains":

# mox.conf

EOF
./mox config describe-static | sed 's/^/\t/'

cat <<EOF

# domains.conf

EOF
./mox config describe-domains | sed 's/^/\t/'

cat <<EOF

# Examples

Mox includes configuration files to illustrate common setups. You can see these
examples with "mox example", and print a specific example with "mox example
<name>". Below are all examples included in mox.

EOF

for ex in $(./mox example); do
	echo '# Example '$ex
	echo
	./mox example $ex | sed 's/^/\t/'
	echo
done

cat <<EOF
*/
package config

// NOTE: DO NOT EDIT, this file is generated by ../gendoc.sh.
EOF
)>config/doc.go
gofmt -w config/doc.go
