# snip

A tool for "snippets", i.e. short notes for things I work on because I can't
remember it otherwise. Useful for seeing where my time goes and what I actually
work on over time.

## Installation

```shell
go install github.com/saser/snip
```

## Usage

Snippets are stored in a directory tree under `~/.snip`.

Record a new snippet:

-   Open an editor to write your snippet:
    ```shell
    snip
    ```
-   Single line:
    ```shell
    snip -m 'finished API draft and sent it out for review'
    ```
-   Start with a single line and open an editor to edit the rest:
    ```shell
    snip -m 'a huge amount of work' -edit
    ```

Snippets are only allowed to contain a single line. Newlines will be replaced with spaces.

TODO: ideas for future use cases:
-   Query snippets for a specific day/week/month/etc
-   Markdown support
-   Some kind of basic web UI?
