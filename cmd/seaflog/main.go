package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/seaflow-uw/seaflog"
	"github.com/urfave/cli/v2"
)

var cmdname string = "seaflog"

func main() {
	app := &cli.App{
		Name:      cmdname,
		Version:   seaflog.Version,
		Usage:     "convert a SeaFlow v1 log file to TSDATA format\n              https://github.com/armbrustlab/tsdataformat",
		UsageText: "seaflog [global options]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "filetype",
				Usage:    "identifier for this file type, no spaces (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "project",
				Usage:    "identifier for this project, no spaces (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "description",
				Usage: "long form file description",
			},
			&cli.StringFlag{
				Name:  "earliest",
				Usage: "RFC3339 timestamp of earliest event to output",
			},
			&cli.StringFlag{
				Name:  "latest",
				Usage: "RFC3339 timestamp of latest event to output",
			},
			&cli.StringFlag{
				Name:     "logfile",
				Usage:    "SeaFLow v1 instrument log file, '-' for STDIN (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "outfile",
				Usage:    "output text file for logfile events in TSDATA format, '-' for STDOUT (required)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "quiet",
				Usage: "don't report parsing errors",
			},
		},
		Action: func(c *cli.Context) (err error) {
			// Parse any timestamps
			earliest := time.Time{}
			latest := time.Time{}
			if c.String("earliest") != "" {
				earliest, err = time.Parse(time.RFC3339, c.String("earliest"))
				if err != nil {
					fmt.Fprintf(c.App.Writer, "error parsing timestamp for --earliest %s\n", c.String("earliest"))
					return err
				}
			}
			if c.String("latest") != "" {
				latest, err = time.Parse(time.RFC3339, c.String("latest"))
				if err != nil {
					fmt.Fprintf(c.App.Writer, "error parsing timestamp for --latest %s\n", c.String("latest"))
					return err
				}
			}

			seaflog.Quiet(c.Bool("quiet"))

			// Open files
			var r *os.File
			var w *os.File
			if c.String("logfile") == "-" {
				r = os.Stdin
			} else {
				r, err = os.Open(c.String("logfile"))
				if err != nil {
					return err
				}
				defer func() {
					if closeErr := r.Close(); closeErr != nil {
						if err == nil {
							err = closeErr
						}
					}
				}()
			}
			bufr := bufio.NewReader(r)
			if c.String("outfile") == "-" {
				w = os.Stdout
			} else {
				if err = os.MkdirAll(filepath.Dir(c.String("outfile")), os.ModePerm); err != nil {
					return err
				}
				w, err = os.Create(c.String("outfile"))
				if err != nil {
					return err
				}
			}
			bufw := bufio.NewWriter(w)
			// Defer flush and close
			defer func() {
				flushErr := bufw.Flush()
				var closeErr error
				if c.String("outfile") != "-" {
					closeErr = w.Close()
				}
				if flushErr != nil || closeErr != nil {
					if err == nil {
						err = fmt.Errorf("flush error: %v, close error: %v", flushErr, closeErr)
					}
				}
			}()

			// Create writer
			tsdw := seaflog.NewTsdataWriter(
				c.String("filetype"), c.String("project"), c.String("description"),
			)
			// Write header
			if _, err := fmt.Fprintf(bufw, "%s\n", tsdw.HeaderText()); err != nil {
				return err
			}
			// Start parsing and write events
			scanner := seaflog.NewEventScanner(bufr)
			for scanner.Scan() {
				event := scanner.Event()
				if !seaflog.TimeFilter(event, earliest, latest) {
					continue
				}
				if event.Name == "unhandled" {
					event = seaflog.UnhandledToNote(event)
					seaflog.Log.Printf(
						"Line %d, unrecognized event, treating as a \"note\".\n  %s\n", event.LineNumber, event.Line,
					)
				}
				if event.Error != nil {
					seaflog.Log.Printf("Line %d, %v.\n  %s\n", event.LineNumber, event.Error, event.Line)
				} else {
					eventLine, err := tsdw.EventText(event)
					if err != nil {
						seaflog.Log.Printf(
							"Line %d, error serializing, %v.\n  %s\n", event.LineNumber, err, event.Line,
						)
					} else {
						if _, err = fmt.Fprintf(bufw, "%s\n", eventLine); err != nil {
							return err
						}
					}
				}
			}
			if err := scanner.Err(); err != nil {
				return err
			}

			return
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
