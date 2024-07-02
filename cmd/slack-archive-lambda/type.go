package main

type archiveRequest struct {
	SlackToken   string   `json:"slack_token"`
	SlackChannel string   `json:"slack_channel"`
	Since        string   `json:"since"`
	Until        string   `json:"until"`
	To           []string `json:"to"`
	Subject      string   `json:"subject"`
	S3Bucket     string   `json:"s3_bucket"`
	S3Key        string   `json:"s3_key"`
}
