# slack-archive

Slackのポストをファイルに書き出すだけ

## Config

Duration config:
- Since SA_SINCE: unixtimestamp
- Until SA_UNTIL: unixtimestamp
- Before SA_BEFORE: unixtimestamp
- After SA_AFTER: unixtime
- Duration SA_DURATION: seconds

since < x < until
x:duration < Before
After < x:duration

Slack Config:
- Token SA_SLACK_TOKEN: string
- ChannelID SA_SLACK_CHANNEL: string

Output Config:
- file
- S3 (TODO)
- mail (TODO)

## CLI Mode

### Get channel history

channel_id string (ex. CXXXXXXXXXX)
before/after ISO8601: (ex. 2024-06-14T15:04:05)

### Get thread hisotry

channel_id string (ex. CXXXXXXXXXX)
thread_ts string (ex. 1718183481.963719)

or

url string (ex. https://examplews.slack.com/archives/CXXXXXXXXXX/p1718243827457659?thread_ts=1718183481.963719&cid=CXXXXXXXXXX
Automaticaly detect channel_id and thread_ts from given URL
(TODO)

## HTTP WebAPI Mode(TODO)

- GET: /
    - Dispatch form
- POST: /run/channel
    - Run with channel mode

```
Header Content-Type: application/json
{
    "channel_id": "CXXXXXXXXXX",
    "before": "2024-06-14T00:00:00",
    "after": "2024-06-14T23:59:59",
}
```

- POST: /run/thread
    - Run with thread mode

```
Header Content-Type: application/json
{
    "channel_id": "CXXXXXXXXXX",
    "thread_ts": "1718183481.963719"
}
```

## Process

### Flow and Components

Collector -> Exporter

- Collector
    - channel: slack.GetConversationHistory
    - thread: GetConversationReplies
- Exporter
    - destination
        - File(io.Writer)
        - S3(TODO)
        - Mail(TODO)
    - Format
        - text
        - json

