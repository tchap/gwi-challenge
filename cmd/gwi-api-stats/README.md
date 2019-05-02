# gwi-api-stats

The executable in this package can be used to query `gwi-api`
to get the current team member counts.

The output format is JSON. It is easy to add more metrics when needed.

## Build

The executable was tested with Go 1.12. Go modules are needed.

## Usage

Once built, the executable can be run in the following way:

```bash
$ ./gwi-api-stats http://$USER:$PASSWORD@$ADDR
```

where

* `USER` is the stats auth username,
* `PASSWORD` is the stats auth password,
* `ADDR` is the base URL where `gwi-api` is listening.

The `out` command line flag can be used to redirect (append) output to a file.

## Example

```bash
$ ./gwi-api-stats http://admin:secret@localhost:8888
{"teams.member_count":{"gophers":1}}
```