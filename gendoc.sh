#!/usr/bin/env sh

(
cat <<EOF
/*
Command mox is a modern, secure, full-featured, open source mail server for
low-maintenance self-hosted email.

Mox is started with the "serve" subcommand, but mox also has many other
subcommands.

Many of those commands talk to a running mox instance, through the ctl file in
the data directory. Specify the configuration file (that holds the path to the
data directory) through the -config flag or MOXCONF environment variable.

Commands that don't talk to a running mox instance are often for
testing/debugging email functionality. For example for parsing an email message,
or looking up SPF/DKIM/DMARC records.

Below is the usage information as printed by the command when started without
any parameters. Followed by the help and usage information for each command.


# Usage

EOF

./mox 2>&1 | sed 's/^\( *\|usage: \)/\t/'

cat <<EOF

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
Package config holds the configuration file definitions.

Mox uses two config files:

1. mox.conf, also called the static configuration file.
2. domains.conf, also called the dynamic configuration file.

The static configuration file is never reloaded during the lifetime of a
running mox instance. After changes to mox.conf, mox must be restarted for the
changes to take effect.

The dynamic configuration file is reloaded automatically when it changes.
If the file contains an error after the change, the reload is aborted and the
previous version remains active.

Below are "empty" config files, generated from the config file definitions in
the source code, along with comments explaining the fields. Fields named "x" are
placeholders for user-chosen map keys.

# sconf

The config files are in "sconf" format. Properties of sconf files:

- Indentation with tabs only.
- "#" as first non-whitespace character makes the line a comment. Lines with a
  value cannot also have a comment.
- Values don't have syntax indicating their type. For example, strings are
  not quoted/escaped and can never span multiple lines.
- Fields that are optional can be left out completely. But the value of an
  optional field may itself have required fields.

See https://pkg.go.dev/github.com/mjl-/sconf for details.


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
