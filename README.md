# slack-archive

Slackのログとファイルをいろんな場所に書き出す


## Usage

### CLI mode

#### environments

```
SA_SLACK_TOKEN=[xoxb-YOURSLACKTOKEN]
SA_SLACK_CHANNEL=[Slack channel ID]

# TextFormatter
SA_TEXT_FORMATTER_REPLY_INDENT_BASE64=ICAgIA==

# Local Exporter
SA_LOCAL_EXPORTER_LOGFILE=/dev/stdout
SA_LOCAL_EXPORTER_FILEDIR=/tmp/slack-archive

# Amazon S3 Exporter
SA_S3_EXPORTER_BUCKET=[S3 Bucket name without s3:// prefix]
SA_S3_EXPORTER_ARCHIVE_FILENAME=[path/to/log-text-file]
SA_S3_EXPORTER_FILES_KEY_PREFIX=[path/to/files/basedir/]

# Amazon SES Exporter
SA_SES_EXPORTER_CONFIG_SET_NAME=[SES Configuration set name]
SA_SES_EXPORTER_SOURCE_ARN=[SES source ARN]
SA_SES_EXPORTER_SUBJECT=[Mail SUBJECT text]
SA_SES_EXPORTER_FROM=[Mail FROM address]
SA_SES_EXPORTER_TO=[Mail TO address]
```

#### command

```shell
go run cmd/slack-archive/main.go \
    --since $(gdate --date '2024-07-01' +%s) \
    --duration 24h \
    --formatter text \
    --text-exporter local \
    --file-exporter local
```

## Lambda Web endpoint

Build `cmd/slack-archive-lambda` as `bootstrap` and Deploy lambda using provided.al2023 runtime

#### Lambda environment

```
SA_SES_EXPORTER_CONFIG_SET_NAME=[SES Configuration set name]
SA_SES_EXPORTER_SOURCE_ARN=[SES source ARN]
```

#### POST request payload

```json
{
    "slack_channel":"Slack channel ID",
    "slack_token":"xoxb-YOURSLACKTOKEN",
    "since":"2024-07-01T12:00:00+09:00",
    "until": "2024-07-02T12:00:00+09:00",
    "To":["receiver.address@example.com"],
    "s3_bucket":"[S3 bucket name]",
    "s3_key": "[path/to/files/basekey/]"
}
```

## Custom Formatter and Exporter

types.goのFormatterInterfaceとTextExporterInterface, FileExporterInterfaceを満たす構造体をConfigに入れることで任意のフォーマットで任意のExport先を追加できます
