package archive

import (
	"context"
	"fmt"
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
	defer func() {
		logger.Printf("Remove %s", config.TempFileDir)
		os.RemoveAll(config.TempFileDir)
	}()

	slackCollectorConfig := NewCollectorSlackConfig(config)
	collector := NewCollectorSlack(slackCollectorConfig, config)

	outputs, err := collector.Execute(ctx)
	if err != nil {
		return err
	}

	formatter := &FormatterText{}
	bytes := formatter.Format(outputs)

	exporter := &ExporterFile{
		Writer: os.Stdout,
	}
	if err := exporter.Write(ctx, bytes); err != nil {
		return err
	}

	return nil
}

type Config struct {
	Since   time.Time
	Until   time.Time
	OutFile string

	TempFileDir string
}

func newConfig() *Config {
	since := flag.Int64("since", 0, "Archive message since")
	until := flag.Int64("until", 0, "Archive message until")
	before := flag.Int64("before", 0, "Archive message before")
	after := flag.Int64("after", 0, "Archive message after")
	duration := flag.String("duration", "", "Archive message duration")
	outfile := flag.StringP("outfile", "o", "", "Output file path")
	tempFileDir := flag.String("temp-file-dir", "", "Temporary file save directory")
	flag.Parse()

	if *tempFileDir == "" {
		d, err := os.MkdirTemp("", fmt.Sprintf("sa_%d", time.Now().Unix()))
		if err != nil {
			panic(err)
		}
		*tempFileDir = d
	}
	conf := &Config{
		OutFile:     *outfile,
		TempFileDir: *tempFileDir,
	}

	if *duration != "" {
		if *before+*after == 0 {
			panic("Duration must be specified along with either 'before' or 'after' flags.")
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
