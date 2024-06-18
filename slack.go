package archive

import (
	"context"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

type CollectorSlackConfig struct {
	Token   string
	Channel string

	HistoryLimit int
	HistryLatest string // archive.Config.Until
	HistryOldest string // archive.Config.since

	RetrivalLimit int
}

func NewCollectorSlackConfig(archiveConf *Config) *CollectorSlackConfig {
	conf := &CollectorSlackConfig{}
	conf.HistoryLimit = 200
	conf.RetrivalLimit = 10

	// Get Environment
	conf.Token = os.Getenv("SA_SLACK_TOKEN")
	conf.Channel = os.Getenv("SA_SLACK_CHANNEL")

	if !archiveConf.Since.IsZero() {
		conf.HistryOldest = strconv.FormatInt(archiveConf.Since.Unix(), 10)
	}
	if !archiveConf.Until.IsZero() {
		conf.HistryLatest = strconv.FormatInt(archiveConf.Until.Unix(), 10)
	}
	return conf
}

// Collector
type CollectorSlack struct {
	slackClient   *slack.Client
	config        *CollectorSlackConfig
	archiveConfig *Config

	// NOTE: slack.MessageのFilesはなぜかSize=0のファイルが飛んでくる
	// messages及びreplyMessagesに入れるタイミングで省くのは難しいので、func getAllFiles()で省き、
	// 取ってきた一覧をtempFilePathsに入れて、func Outputs() のタイミングで存在するものだけOutputsに
	// 入れて返す形になっている
	messages      []slack.Message
	replyMessages map[string][]slack.Message
	tempFilePaths map[string]string
	userCache     *userCacheClient
}

// Interface implementation check
var _ CollectorInterface = (*CollectorSlack)(nil)

func NewCollectorSlack(conf *CollectorSlackConfig, aConf *Config) *CollectorSlack {
	return &CollectorSlack{
		slackClient:   slack.New(conf.Token),
		config:        conf,
		archiveConfig: aConf,
		messages:      []slack.Message{},
		replyMessages: map[string][]slack.Message{},
		tempFilePaths: map[string]string{},
		userCache:     newUserCacheClient(),
	}
}

func (collector *CollectorSlack) Execute(ctx context.Context) (Outputs, error) {
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

func (collector *CollectorSlack) getHistoryMessages(ctx context.Context) error {
	client := collector.slackClient
	config := collector.config

	var messages = []slack.Message{}

	// conversations.history
	var cur string = ""
	var count = 0
	for count < config.RetrivalLimit {
		count++
		logger.Printf("GetConversationHistoryContext count:%d\n", count)

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

func (collector *CollectorSlack) getHistoryMessagesInThread(ctx context.Context) error {
	client := collector.slackClient
	config := collector.config

	var threadBaseMessages = []slack.Message{}
	for _, msg := range collector.messages {
		logger.Printf("collector.messages ts:%s subtype:%s replyCount:%d", msg.Timestamp, msg.SubType, msg.ReplyCount)
		if msg.ReplyCount != 0 {
			threadBaseMessages = append(threadBaseMessages, msg)
		}
	}

	// conversations.replies
	for _, baseMsg := range threadBaseMessages {
		if _, ok := collector.replyMessages[baseMsg.Timestamp]; !ok {
			collector.replyMessages[baseMsg.Timestamp] = []slack.Message{}
		}
		logger.Printf("ReplyMessages %s", baseMsg.Timestamp)

		var cur string = ""
		var count = 0
		for count < config.RetrivalLimit {
			count++
			logger.Printf("GetConversationHistoryContext base: %s, count:%d\n", baseMsg.Timestamp, count)

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

func (collector *CollectorSlack) getUserdata(ctx context.Context) error {
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

func (collector *CollectorSlack) getUsername(ctx context.Context, uid string) (string, error) {
	logger.Printf("GetUserProfileContext UserID:%s\n", uid)
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

func (collector *CollectorSlack) outputs() (Outputs, error) {
	var outputs Outputs

	for _, msg := range collector.messages {
		if msg.SubType == "thread_broadcast" {
			continue
		}

		output, err := collector.slackMessageToOutput(msg)
		if err != nil {
			logger.Printf("failed to convert slackMessage to archive.Output: %s", err)
			continue
		}

		// リプライがある場合は後ろにくっつける
		if replies, ok := collector.replyMessages[msg.Timestamp]; ok {
			outputReplies := Outputs{}
			for _, reply := range replies {
				outputReply, err := collector.slackMessageToOutput(reply)
				if err != nil {
					logger.Printf("failed to convert slackMessage to archive.Output: %s", err)
					continue
				}
				// NOTE: Repliesにはスレッドの元になるポストが含まれるのでスキップする
				if msg.Timestamp == reply.Timestamp {
					continue
				}
				outputReplies = append(outputReplies, outputReply)
			}
			logger.Printf("Replies %d\n", len(outputReplies))
			output.Replies = outputReplies
		}

		outputs = append(outputs, output)
	}

	return outputs, nil
}

func (collector *CollectorSlack) slackMessageToOutput(msg slack.Message) (*Output, error) {
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
	files := []*TempFile{}
	for _, slackFile := range msg.Files {
		tempPath, ok := collector.tempFilePaths[slackFile.ID]
		if !ok {
			continue
		}
		f := &TempFile{
			id:        slackFile.ID,
			timestamp: slackFile.Timestamp.Time(),
			name:      slackFile.Name,
			path:      tempPath,
		}
		files = append(files, f)
	}

	return &Output{
		Timestamp: timestamp,
		Username:  displayName,
		Text:      text,
		TempFiles: files,
	}, nil
}

func (collector *CollectorSlack) getAllFiles(ctx context.Context) error {
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

func (collector *CollectorSlack) getFileAndPutTemporaryPath(ctx context.Context, slackFile slack.File) (string, error) {
	path := path.Join(collector.archiveConfig.TempFileDir, slackFile.ID)
	if _, ok := collector.tempFilePaths[slackFile.ID]; ok {
		// Already downloaded
		return path, nil
	}

	logger.Printf("Save temporary file %s(%dbyte) -> %s (%s)", slackFile.Name, slackFile.Size, path, slackFile.Filetype)
	f, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	collector.slackClient.GetFileContext(ctx, slackFile.URLPrivate, f)
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

func (collector *CollectorSlack) userdataFetchAll(ctx context.Context) error {
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
