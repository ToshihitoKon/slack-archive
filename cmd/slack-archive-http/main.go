package main

import (
	// "context"
	"log"
	"net/http"
	// archive "github.com/ToshihitoKon/slack-archive"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/slack/channel", handler)
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	// ctx := context.Background()

	// archiveConf := &archive.Config{}
	// slackConf := archive.NewCollectorSlackConfig(archiveConf)
	// collector := archive.NewCollectorSlack(slackConf, archiveConf)

	// outputs, err := collector.Execute(ctx)
	// if err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// }
	//
	// formatter := &archive.FormatterText{}
	// byte := formatter.Format(outputs)
	//
	// exporter := &archive.ExporterFile{
	// 	Writer: w,
	// }
	// if err := exporter.Write(ctx, byte); err != nil {
	// 	w.WriteHeader(http.StatusInternalServerError)
	// }
	w.WriteHeader(http.StatusOK)
}
