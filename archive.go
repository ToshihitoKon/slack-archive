package archive

import (
	"context"
	"log"
	"os"
	"time"

	flag "github.com/spf13/pflag"
)

var logger = log.New(os.Stderr, "[info]", log.Lshortfile)

func Run() error {
	ctx := context.Background()
	return run(ctx)
}

func run(ctx context.Context) error {
	config := newConfig()
	// logger.Printf("%+v", config)
	collector := NewArchiveCollectorSlack(config)

	outputs, err := collector.Execute(ctx)
	if err != nil {
		return err
	}

	exporter := &ArchiveExporterString{}
	if err := exporter.Write(ctx, outputs, os.Stdout); err != nil {
		return err
	}

	return nil
}

type archiveConfig struct {
	Since time.Time
	Until time.Time

	OutFile string
}

func newConfig() *archiveConfig {
	since := flag.Int64("since", 0, "Archive message since")
	until := flag.Int64("until", 0, "Archive message until")
	before := flag.Int64("before", 0, "Archive message before")
	after := flag.Int64("after", 0, "Archive message after")
	duration := flag.String("duration", "", "Archive message duration")
	outfile := flag.StringP("outfile", "o", "", "Output file path")
	flag.Parse()

	conf := &archiveConfig{
		OutFile: *outfile,
	}

	if *duration != "" {
		if *before+*after == 0 {
			panic("duration must be specify with before or after")
		}
		if *since+*until != 0 {
			panic("since or until can't specify with duration")
		}

		dur, err := time.ParseDuration(*duration)
		if err != nil {
			panic(err)
		}
		switch {
		case *before != 0:
			conf.Until = time.Unix(*before, 0)
			conf.Since = conf.Until.Add(dur * -1)
		case *after != 0:
			conf.Since = time.Unix(*after, 0)
			conf.Until = conf.Since.Add(dur)
		}
	} else {
		if *since != 0 {
			conf.Since = time.Unix(*since, 0)
		}
		if *until != 0 {
			conf.Until = time.Unix(*until, 0)
		}
	}
	return conf
}
