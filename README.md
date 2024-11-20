# `snip`

`snip` is a simple tool for recording "snippets", i.e. short notes with a
timestamp intended to record what I'm currently doing so that I can review it later.

The notes are stored as plain text files in `~/.snip/YYYY-MM-DD.txt`. Each file
contains the snippets recorded on that date, one snippet per line, in the order
they were recorded.
```
$ tree ~/.snip
/Users/saser/.snip
├── 2024-11-15.txt
├── 2024-11-18.txt
├── 2024-11-20.txt
...
```

## Purpose and example workflow

I use it primarily in my workday to record what I'm working on, because I have
more concurrent workstreams than I can keep in my head. The snippets help me
what I'm working on (which is not often what I think I'm working on!) as well as
how much time I spend on it.

I've also found snippets useful for things like performance reviews, when I need
to summarize what I've worked on in the last quarter(s) or so. By recording
little notes here and there, and reviewing them on some regular basis, I get a
much clearer picture. I have often found that I did some valuable piece of work
that I have forgotten about!

(In reality, I use shell aliases to save me some typing. See the Aliases section
below.)

In the future I might add some basic querying capabilities to `snip` itself,
something like `snip last 3 days`, `snip this week`, etc.

## Installation

```
go install github.com/saser/snip
```

## Usage

Example of my workflow for recording notes:
```
$ snip -m "at desk; going to review Alice's MR"
$ snip -m 'reviewed the MR; now going to start working on the system design draft'
$ snip -m 'got sidetracked into a debugging session; continuing work on the draft'
$ snip -m 'heading for lunch'
$ snip -m 'back from lunch; heading into 1:1 with mgr'
# etc
```
These notes are stored in `~/.snip/2024-11-20.txt` as:
```
2024-11-20 09:30:47 +0000 | at desk; going to review Alice's MR
2024-11-20 09:53:24 +0000 | reviewed the MR; now going to start working on the system design draft
2024-11-20 10:50:43 +0000 | got sidetracked into a debugging session; continuing work on the draft
2024-11-20 12:09:04 +0000 | heading for lunch
2024-11-20 13:38:00 +0000 | back from lunch + coffee walk; heading into 1:1 with mgr
```
(The date/timestamps are just examples, of course.)

Invoking `snip` without any arguments or flags will open an editor to write your
note in a temporary file.
```
$ snip
```
`snip` will prepopulate the file with the current timestamp:
```
2024-11-20 16:30:56 +0000 | <write your note here>
```
`snip` will use `$EDITOR` if it's set; fall back to `vim` if it's not; and
exit with an error if `vim` isn't in `$PATH`:

Timestamps are not mandatory, they're just added there for convenience. Remove
them if you don't want them. The only requirement is that the snippet is not
empty. Note that snippets are intended to be single lines; newlines will be
replaced by spaces.

To avoid a roundtrip to the editor, use the `-m` flag. Note that the flag takes
a single string as an argument, so use quotes in your shell.
```
$ snip -m 'worked on the API draft'
```

If using `-m` but realize you want to open an editor, add the `-edit` flag.
```
$ snip -m 'started working on the architecture document but' -edit
```
`snip` will take whatever you have written in the `-m` flag so far and
prepopulate it in the editor together with the current time:
```
2024-11-20 16:30:56 +0000 | started working on the architecture document but
```

## Flexibility

Like mentioned above, snippets recorded by `snip` are stored in text files as
`~/.snip/YYYY-MM-DD.txt`:
```
$ tree ~/.snip
/Users/saser/.snip
├── 2024-11-15.txt
├── 2024-11-18.txt
├── 2024-11-20.txt
...
```
Since they're just text files, and `snip` doesn't parse their contents (yet...),
you have a fair amount of flexibility to adapt `snip` to different workflows.

For example, if you want to tag your snippets
with keywords to track specific projects or something, you can adopt the
convention to add `#foo` for everything related to the `foo` project, and then
`grep` for all such tags:
```
$ grep '#foo' ~/.snip/*.txt
/Users/saser/.snip/2024-11-15.txt:2024-11-15 14:49:49 +0000 | asked Alice about using Prometheus for metrics #foo
/Users/saser/.snip/2024-11-15.txt:2024-11-15 15:57:48 +0000 | got some basic metrics exporting working in test dev!! yay #foo
/Users/saser/.snip/2024-11-15.txt:2024-11-15 16:30:48 +0000 | started working on integrating Telegraf setup into Ansible playbooks; made some progress but still lots that needs fleshing out, e.g. getting secrets from Vault. headed to standup #foo
/Users/saser/.snip/2024-11-18.txt:2024-11-18 11:16:04 +0000 | got roped into some AWS cost analysis, doesn't seem like our team is contributing to any big costs. Continuing with benchmarks #foo
/Users/saser/.snip/2024-11-18.txt:2024-11-18 14:16:33 +0000 | still running benchmarks... realized I didn't try doing parallel requests #foo
/Users/saser/.snip/2024-11-18.txt:2024-11-18 15:09:20 +0000 | not sure exactly where time has gone, but I have JIRA-123 for rolling out the new page sizes; still needs the deployment MR but I've sent it for review to Alice and Bob #foo
/Users/saser/.snip/2024-11-20.txt:2024-11-20 12:09:04 +0000 | back from demo presentation, now heading straight to lunch #foo
```

## Aliases

In my personal setup I use some shell aliases to make it a bit easier and faster
to record and view snippets. Below is a dump of my current aliases (on macOS):
```shell
alias s='snip'
alias sm='snip -m'
alias sme='snip -edit -m'
alias st='cat ~/.snip/$(date -I).txt'   # "st" = "snip today"
alias ste='vim ~/.snip/$(date -I).txt'  # "ste" = "snip today edit"
```
I use `s` and `sm` the most.

## TODO

Ideas for future use cases:
-   Query snippets for a specific day/week/month/etc
-   Some kind of basic web UI? w/ Markdown support?
