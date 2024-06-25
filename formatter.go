package archive

import (
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
)

type TextFormatter struct {
	ReplyIndent string
}

var _ FormatterInterface = (*TextFormatter)(nil)

func NewTextFormatter() *TextFormatter {
	replyIndent := os.Getenv("SA_TEXT_FORMATTER_REPLY_INDENT")
	if replyIndent == "" {
		replyIndent = " | "
	}
	return &TextFormatter{
		ReplyIndent: replyIndent,
	}
}

func (f *TextFormatter) WriteFileName(file *LocalFile) string {
	return path.Join(fmt.Sprintf("%s_%s", file.id, file.name))
}
func (f *TextFormatter) Format(outputs Outputs) []byte {
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Timestamp.Before(outputs[j].Timestamp) })

	texts := []string{}
	for _, output := range outputs {
		text := fmt.Sprintf(
			"[%s] [%s] %s",
			output.Timestamp.Format("2006/01/02 15:04:05"),
			output.Username,
			output.Text,
		)
		for _, tfile := range output.LocalFiles {
			text += fmt.Sprintf("\n(file: %s)", f.WriteFileName(tfile))
		}
		texts = append(texts, text)

		// replies
		sort.Slice(output.Replies, func(i, j int) bool { return output.Replies[i].Timestamp.Before(output.Replies[j].Timestamp) })
		for _, reply := range output.Replies {
			textBody := strings.ReplaceAll(reply.Text, "\n", fmt.Sprintf("\n%s", f.ReplyIndent))
			text := fmt.Sprintf(
				"%s[%s] [%s] %s",
				f.ReplyIndent,
				reply.Timestamp.Format("2006/01/02 15:04:05"),
				reply.Username,
				textBody,
			)
			for _, tfile := range reply.LocalFiles {
				text += fmt.Sprintf("\n%s(file: %s)", f.ReplyIndent, f.WriteFileName(tfile))
			}
			texts = append(texts, text)
		}
	}
	return []byte(strings.Join(texts, "\n"))
}
