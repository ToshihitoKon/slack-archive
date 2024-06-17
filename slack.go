package archive

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/slack-go/slack"
)

type slackConfig struct {
	slackToken        string
	slackHistoryLimit int
	slackChannel      string

	slackHistryLatest string // archiveConfig.Until
	slackHistryOldest string // archiveConfig.since

	retrivalLimit int
}

func (conf *slackConfig) init(aConf *archiveConfig) {
	conf.slackHistoryLimit = 200
	conf.retrivalLimit = 10

	// Get Environment
	conf.slackToken = os.Getenv("SA_SLACK_TOKEN")
	conf.slackChannel = os.Getenv("SA_SLACK_CHANNEL")

	if !aConf.Since.IsZero() {
		conf.slackHistryOldest = strconv.FormatInt(aConf.Since.Unix(), 10)
	}
	if !aConf.Until.IsZero() {
		conf.slackHistryLatest = strconv.FormatInt(aConf.Until.Unix(), 10)
	}
}

// Collector
type ArchiveCollectorSlack struct {
	slackClient   *slack.Client
	config        *slackConfig
	archiveConfig *archiveConfig

	messages  []slack.Message
	userCache *userCacheClient
}

// Interface implementation check
var _ ArchiveCollectorInterface = (*ArchiveCollectorSlack)(nil)

func NewArchiveCollectorSlack(aConf *archiveConfig) *ArchiveCollectorSlack {
	conf := &slackConfig{}
	conf.init(aConf)

	return &ArchiveCollectorSlack{
		slackClient:   slack.New(conf.slackToken),
		config:        conf,
		archiveConfig: aConf,
		messages:      []slack.Message{},
		userCache:     NewUserCacheClient(),
	}
}

func (collector *ArchiveCollectorSlack) Execute(ctx context.Context) ([]*ArchiveOutput, error) {
	if err := collector.getMessages(ctx); err != nil {
		return nil, err
	}

	if err := collector.getUserdata(ctx); err != nil {
		return nil, err
	}

	outputs, err := collector.archiveOutputs()
	if err != nil {
		return nil, err
	}

	return outputs, nil
}

func (collector *ArchiveCollectorSlack) getMessages(ctx context.Context) error {
	client := collector.slackClient
	config := collector.config

	var messages = []slack.Message{}

	// conversations.history
	var cur string = ""
	var count = 0
	for count < config.retrivalLimit {
		count++
		logger.Printf("GetConversationHistoryContext count:%d\n", count)

		params := &slack.GetConversationHistoryParameters{
			ChannelID:          config.slackChannel,
			Cursor:             cur,
			Limit:              config.slackHistoryLimit,
			IncludeAllMetadata: false,
		}
		if config.slackHistryLatest != "" {
			params.Latest = config.slackHistryLatest
		}
		if config.slackHistryOldest != "" {
			params.Oldest = config.slackHistryOldest
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

func (collector *ArchiveCollectorSlack) getUserdata(ctx context.Context) error {
	// Users map
	uids := []string{}
	for _, msg := range collector.messages {
		uids = append(uids, msg.User)
		uids = append(uids, msg.ReplyUsers...)
		// NOTE: リアクションのアーカイブ非対応なのでユーザーID検索もスキップ
		// for _, r := range msg.Reactions {
		// 	uids = append(uids, r.Users...)
		// }
	}

	for _, uid := range uids {
		// NOTE: PutIfNotExistにfunc(string)stringを渡せば中でslackAPIを叩かせられるけど面倒になって一旦保留
		// userdataFetchAllで一気に叩く
		collector.userCache.PutIfNotExist(uid, "")
	}

	if err := collector.userdataFetchAll(ctx); err != nil {
		return err
	}

	return nil
}

func (collector *ArchiveCollectorSlack) getUsername(ctx context.Context, uid string) (string, error) {
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

func (collector *ArchiveCollectorSlack) archiveOutputs() ([]*ArchiveOutput, error) {
	var outputs []*ArchiveOutput

	for _, msg := range collector.messages {
		if msg.SubType == "thread_broadcast" {
			continue
		}
		displayName, ok := collector.userCache.cache[msg.User]
		if !ok {
			displayName = msg.User
		}

		tsMicro, err := strconv.ParseInt(strings.ReplaceAll(msg.Timestamp, ".", ""), 10, 64)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to parseint: %s", msg.Timestamp)
			continue
		}
		timestamp := time.UnixMicro(tsMicro)
		text := collector.userCache.ReplaceAll(msg.Text)

		outputs = append(outputs, &ArchiveOutput{
			Timestamp: timestamp,
			Username:  displayName,
			Text:      text,
		})
	}
	return outputs, nil
}

func NewUserCacheClient() *userCacheClient {
	return &userCacheClient{
		cache: map[string]string{},
	}
}

func (ucc *userCacheClient) PutIfNotExist(key, value string) {
	if _, ok := ucc.cache[key]; !ok {
		ucc.cache[key] = value
	}
}

func (ucc *userCacheClient) ReplaceAll(str string) string {
	result := str
	for uid, name := range ucc.cache {
		result = strings.ReplaceAll(result, uid, name)
	}
	return result
}

func (collector *ArchiveCollectorSlack) userdataFetchAll(ctx context.Context) error {
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
