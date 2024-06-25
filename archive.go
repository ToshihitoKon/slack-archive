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
		logger.Printf("Remove %s", config.LocalFileDir)
		os.RemoveAll(config.LocalFileDir)
	}()

	slackCollectorConfig := NewSlackCollectorConfig(config)
	collector := NewSlackCollector(slackCollectorConfig, config)

	var formatter FormatterInterface
	switch config.Format {
	case "text":
		formatter = &TextFormatter{}
	default:
		return fmt.Errorf("Format %s is not available", config.Exporter)
	}

	var textExporter TextExporterInterface
	var fileExporter FileExporterInterface
	switch config.Exporter {
	case "none":
		exp := &NoneExporter{}
		textExporter = exp
		fileExporter = exp
	case "file":
		exp := NewLocalExporter()
		textExporter = exp
		fileExporter = exp
	case "s3":
		exp, err := NewS3Exporter(ctx)
		if err != nil {
			return err
		}
		textExporter = exp
		fileExporter = exp
	case "ses":
		tExp, err := NewSESTextExporter(ctx)
		if err != nil {
			return err
		}
		textExporter = tExp

		fExp, err := NewS3Exporter(ctx)
		if err != nil {
			return err
		}
		fileExporter = fExp
	default:
		return fmt.Errorf("Exporter %s is not available", config.Exporter)
	}

	outputs, err := collector.Execute(ctx)
	if err != nil {
		return err
	}

	bytes := formatter.Format(outputs)

	if err := textExporter.Write(ctx, bytes); err != nil {
		return err
	}
	if err := fileExporter.WriteFiles(ctx, outputs.LocalFiles(), formatter.WriteFileName); err != nil {
		return err
	}

	return nil
}

type Config struct {
	Since    time.Time
	Until    time.Time
	Exporter string
	Format   string

	LocalFileDir string
}

func newConfig() *Config {
	since := flag.Int64("since", 0, "Archive message since")
	until := flag.Int64("until", 0, "Archive message until")
	duration := flag.String("duration", "", "Archive message duration")
	format := flag.String("format", "text", "Log format default: text")
	exporter := flag.String("exporter", "file", "Exporter default: file")
	flag.Parse()

	conf := &Config{
		Format:   *format,
		Exporter: *exporter,
	}

	d, err := os.MkdirTemp("", fmt.Sprintf("sa_%d", time.Now().Unix()))
	if err != nil {
		panic(err)
	}
	conf.LocalFileDir = d

	if *duration != "" {
		if *since != 0 && *until != 0 {
			panic("You can't specify both since and until when specify duration")
		}
		dur, err := time.ParseDuration(*duration)
		if err != nil {
			panic(err)
		}

		switch {
		case *since != 0:
			conf.Since = time.Unix(*since, 0)
			conf.Until = conf.Since.Add(dur * 1)
		case *until != 0:
			conf.Until = time.Unix(*until, 0)
			conf.Since = conf.Until.Add(dur * -1)
		default:
			conf.Until = time.Now()
			conf.Since = conf.Until.Add(dur * -1)
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
