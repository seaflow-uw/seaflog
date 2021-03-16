# seaflog

This package provides a command-line tool to parse SeaFlow v1 instrument log
files as [TSDATA format files](https://github.com/armbrustlab/tsdataformat).

## Download binaries

Binaries for Linux and MacOS on x86_64 platforms are available from the releases
page of this GitHub repo.

## Build

This code requires Go 1.16 to build.

Using `go install` in Go 1.16+

```sh
go install github.com/seaflog-uw/seaflog@latest
```

From a local copy of this repo:

```sh
go build -o seaflog cmd/seaflog/main.go
```

## Usage

```sh
seaflog \
    --filetype SeaFlowV1InstrumentLog \
    --project SeaFlow_740 --description "SeaFlow V1 Instrument Log events" \
    --logfile SFlog_740.txt \
    --outfile SFlog_740.tsv
```

See the output of `seaflog --help` for full usage.
