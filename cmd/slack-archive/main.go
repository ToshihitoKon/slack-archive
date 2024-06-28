package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	archive "github.com/ToshihitoKon/slack-archive"
)

func main() {
	conf := parseFlags()
	if err := archive.Run(conf); err != nil {
		log.Fatal(err)
	}
}

func parseFlags() *archive.Config {
	since := flag.Int64("since", 0, "Archive message since")
	until := flag.Int64("until", 0, "Archive message until")
	duration := flag.String("duration", "", "Archive message duration")
	formatter := flag.String("formatter", "text", "Log format default: text")
	exporter := flag.String("exporter", "local", "Exporter default: local")
	flag.Parse()

	conf := &archive.Config{
		Formatter: *formatter,
		Exporter:  *exporter,
		Logger:    slog.Default(),
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
