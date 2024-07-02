package archive

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

type SlackCollectorConfig struct {
	Token   string
	Channel string

	HistoryLimit int
	HistryLatest string // archive.Config.Until
	HistryOldest string // archive.Config.since

	RetrivalLimit int
}

func NewSlackCollectorConfig(archiveConf *Config) *SlackCollectorConfig {
	conf := &SlackCollectorConfig{}
	conf.HistoryLimit = 200
	conf.RetrivalLimit = 10

	conf.Token = firstString([]string{
		archiveConf.SlackToken,
		os.Getenv("SA_SLACK_TOKEN"),
	})
	conf.Channel = firstString([]string{
		archiveConf.SlackChannel,
		os.Getenv("SA_SLACK_CHANNEL"),
	})

	if !archiveConf.Since.IsZero() {
		conf.HistryOldest = strconv.FormatInt(archiveConf.Since.Unix(), 10)
	}
	if !archiveConf.Until.IsZero() {
		conf.HistryLatest = strconv.FormatInt(archiveConf.Until.Unix(), 10)
	}
	return conf
}

// Collector
type SlackCollector struct {
	config        *SlackCollectorConfig
	archiveConfig *Config
	slackClient   *slack.Client

	userCache     *userCacheClient
	messages      []slack.Message
	replyMessages map[string][]slack.Message

	// NOTE: slack.MessageのFilesはなぜかSize=0のファイルが飛んでくる
	// messages及びreplyMessagesに入れるタイミングで省くのは難しいので、func getAllFiles()で省き、
	// 取ってきた一覧をtempFilePathsに入れて、func Outputs() のタイミングで存在するものだけOutputsに
	// 入れて返す形になっている
	tempFileDir   string
	tempFilePaths map[string]string

	logger *slog.Logger
}

// Interface implementation check
var _ CollectorInterface = (*SlackCollector)(nil)

func NewSlackCollector(conf *Config, slackConf *SlackCollectorConfig) *SlackCollector {
	tempFileDirPath, err := os.MkdirTemp("", fmt.Sprintf("sa_%d", time.Now().Unix()))
	if err != nil {
		conf.Logger.Error("os.MkdirTemp failed", "function", "NewSlackCollector", "error", err.Error())
		panic(err)
	}

	return &SlackCollector{
		config:        slackConf,
		archiveConfig: conf,
		slackClient:   slack.New(slackConf.Token),

		userCache:     newUserCacheClient(),
		messages:      []slack.Message{},
		replyMessages: map[string][]slack.Message{},

		tempFileDir:   tempFileDirPath,
		tempFilePaths: map[string]string{},

		logger: conf.Logger,
	}
}

func (collector *SlackCollector) Clean() {
	if err := os.RemoveAll(collector.tempFileDir); err != nil {
		collector.logger.Error("an error occurred", "function", "os.RemoveAll", "error", err.Error())
	}
}

func (collector *SlackCollector) Execute(ctx context.Context) (Outputs, error) {
	if err := collector.getHistoryMessages(ctx); err != nil {
		return nil, err
	}

	if err := collector.getHistoryMessagesInThread(ctx); err != nil {
		return nil, err
	}

	if err := collector.getUserdata(ctx); err != nil {
		return nil, err
	}

	if err := collector.getAllFiles(ctx); err != nil {
		return nil, err
	}

	outputs, err := collector.outputs()
	if err != nil {
		return nil, err
	}

	return outputs, nil
}

func (collector *SlackCollector) getHistoryMessages(ctx context.Context) error {
	client := collector.slackClient
	config := collector.config

	var messages = []slack.Message{}

	// conversations.history
	var cur string = ""
	var count = 0
	for count < config.RetrivalLimit {
		count++
		collector.logger.Info("GetConversationHistoryContext", "count", count)

		params := &slack.GetConversationHistoryParameters{
			ChannelID:          config.Channel,
			Cursor:             cur,
			Limit:              config.HistoryLimit,
			IncludeAllMetadata: false,
		}
		if config.HistryLatest != "" {
			params.Latest = config.HistryLatest
		}
		if config.HistryOldest != "" {
			params.Oldest = config.HistryOldest
		}

		historyRes, err := client.GetConversationHistoryContext(ctx, params)
		if err != nil {
			return err
		}
		if !historyRes.Ok {
			return fmt.Errorf("Slack error: %w, %+v", historyRes.Err(), historyRes.ResponseMetadata)
		}
		messages = append(messages, historyRes.Messages...)

		if !historyRes.HasMore {
			break
		}
		cur = historyRes.ResponseMetaData.NextCursor
	}

	collector.messages = messages
	return nil
}

func (collector *SlackCollector) getHistoryMessagesInThread(ctx context.Context) error {
	client := collector.slackClient
	config := collector.config

	var threadBaseMessages = []slack.Message{}
	for _, msg := range collector.messages {
		collector.logger.Info("collector.messages", "ts", msg.Timestamp, "subtype", msg.SubType, "replyCount", msg.ReplyCount)
		if msg.ReplyCount != 0 {
			threadBaseMessages = append(threadBaseMessages, msg)
		}
	}

	// conversations.replies
	for _, baseMsg := range threadBaseMessages {
		if _, ok := collector.replyMessages[baseMsg.Timestamp]; !ok {
			collector.replyMessages[baseMsg.Timestamp] = []slack.Message{}
		}
		collector.logger.Info("ReplyMessages", "ts", baseMsg.Timestamp)

		var cur string = ""
		var count = 0
		for count < config.RetrivalLimit {
			count++
			collector.logger.Info("GetConversationHistoryContext", "baseMessage timestamp", baseMsg.Timestamp, "count", count)

			params := &slack.GetConversationRepliesParameters{
				ChannelID:          config.Channel,
				Timestamp:          baseMsg.Timestamp,
				Cursor:             cur,
				Limit:              config.HistoryLimit,
				IncludeAllMetadata: false,
			}
			if config.HistryLatest != "" {
				params.Latest = config.HistryLatest
			}
			if config.HistryOldest != "" {
				params.Oldest = config.HistryOldest
			}

			msgs, hasMore, nextCursor, err := client.GetConversationRepliesContext(ctx, params)
			if err != nil {
				return err
			}

			collector.replyMessages[baseMsg.Timestamp] = append(collector.replyMessages[baseMsg.Timestamp], msgs...)

			if !hasMore {
				break
			}
			cur = nextCursor
		}
	}

	return nil
}

func (collector *SlackCollector) getUserdata(ctx context.Context) error {
	for _, msg := range collector.messages {
		collector.userCache.putIfNotExist(msg.User, "")
		// NOTE: リアクションのアーカイブ非対応なのでユーザーID検索もスキップ
		// for _, r := range msg.Reactions {
		// }
	}
	for _, msgs := range collector.replyMessages {
		for _, msg := range msgs {
			collector.userCache.putIfNotExist(msg.User, "")
			// NOTE: リアクションのアーカイブ非対応なのでユーザーID検索もスキップ
			// for _, r := range msg.Reactions {
			// }
		}
	}

	if err := collector.userdataFetchAll(ctx); err != nil {
		return err
	}

	return nil
}

func (collector *SlackCollector) getUsername(ctx context.Context, uid string) (string, error) {
	collector.logger.Info("GetUserProfileContext", "UserID", uid)
	uprof, err := collector.slackClient.GetUserProfileContext(ctx, &slack.GetUserProfileParameters{
		UserID:        uid,
		IncludeLabels: false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to GetUserProfile(%s): %w", uid, err)
	}
	return uprof.DisplayName, nil
}

type userCacheClient struct {
	cache map[string]string
}

func (collector *SlackCollector) outputs() (Outputs, error) {
	var outputs Outputs

	for _, msg := range collector.messages {
		if msg.SubType == "thread_broadcast" {
			continue
		}

		output, err := collector.slackMessageToOutput(msg)
		if err != nil {
			collector.logger.Error("failed to convert slackMessage to archive.Output", "error", err)
			continue
		}

		// リプライがある場合は後ろにくっつける
		if replies, ok := collector.replyMessages[msg.Timestamp]; ok {
			outputReplies := Outputs{}
			for _, reply := range replies {
				outputReply, err := collector.slackMessageToOutput(reply)
				if err != nil {
					collector.logger.Error("failed to convert slackMessage to archive.Output", "error", err)
					continue
				}
				// NOTE: Repliesにはスレッドの元になるポストが含まれるのでスキップする
				if msg.Timestamp == reply.Timestamp {
					continue
				}
				outputReplies = append(outputReplies, outputReply)
			}
			collector.logger.Info("Replies", "count", len(outputReplies))
			output.Replies = outputReplies
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

func (collector *SlackCollector) slackMessageToOutput(msg slack.Message) (*Output, error) {
	var displayName string
	if msg.Username != "" {
		displayName = msg.Username
	} else {
		var ok bool
		displayName, ok = collector.userCache.cache[msg.User]
		if !ok {
			displayName = msg.User
		}
	}

	tsMicro, err := strconv.ParseInt(strings.ReplaceAll(msg.Timestamp, ".", ""), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to ParseInt: %s", msg.Timestamp)
	}
	timestamp := time.UnixMicro(tsMicro)
	text := collector.userCache.replaceAll(msg.Text)

	// Attachment Files
	files := []*LocalFile{}
	for _, slackFile := range msg.Files {
		tempPath, ok := collector.tempFilePaths[slackFile.ID]
		if !ok {
			continue
		}
		f := &LocalFile{
			id:        slackFile.ID,
			timestamp: slackFile.Timestamp.Time(),
			name:      slackFile.Name,
			path:      tempPath,
		}
		files = append(files, f)
	}

	return &Output{
		Timestamp:  timestamp,
		Username:   displayName,
		Text:       text,
		LocalFiles: files,
	}, nil
}

func (collector *SlackCollector) getAllFiles(ctx context.Context) error {
	files := []slack.File{}
	for _, msg := range collector.messages {
		for _, f := range msg.Files {
			if f.Size == 0 {
				continue
			}
			files = append(files, f)
		}
	}
	for _, msgs := range collector.replyMessages {
		for _, msg := range msgs {
			for _, f := range msg.Files {
				if f.Size == 0 {
					continue
				}
				files = append(files, f)
			}
		}
	}

	for _, f := range files {
		p, err := collector.getFileAndPutTemporaryPath(ctx, f)
		if err != nil {
			return err
		}
		collector.tempFilePaths[f.ID] = p
	}
	return nil
}

func (collector *SlackCollector) getFileAndPutTemporaryPath(ctx context.Context, slackFile slack.File) (string, error) {
	path := path.Join(collector.tempFileDir, slackFile.ID)
	if _, ok := collector.tempFilePaths[slackFile.ID]; ok {
		// Already downloaded
		return path, nil
	}

	collector.logger.Info("Save temporary file", "name", slackFile.Name, "destination", "size", path, slackFile.Size, "filetype", slackFile.Filetype)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if err := collector.slackClient.GetFileContext(ctx, slackFile.URLPrivate, f); err != nil {
		return "", err
	}
	return path, nil
}

func newUserCacheClient() *userCacheClient {
	return &userCacheClient{
		cache: map[string]string{},
	}
}

func (ucc *userCacheClient) putIfNotExist(key, value string) {
	if _, ok := ucc.cache[key]; !ok {
		ucc.cache[key] = value
	}
}

func (ucc *userCacheClient) replaceAll(str string) string {
	result := str
	for uid, name := range ucc.cache {
		result = strings.ReplaceAll(result, uid, name)
	}
	return result
}

func (collector *SlackCollector) userdataFetchAll(ctx context.Context) error {
	for uid, name := range collector.userCache.cache {
		if name == "" {
			displayName, err := collector.getUsername(ctx, uid)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				collector.userCache.cache[uid] = uid
				continue
			}
			collector.userCache.cache[uid] = displayName
		}
	}
	return nil
}
