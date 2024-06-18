package archive

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

// Implementations
type FormatterText struct{}

var _ FormatterInterface = (*FormatterText)(nil)

func (f *FormatterText) WriteFileName(file *TempFile) string {
	return path.Join(fmt.Sprintf("%s_%s", file.id, file.name))
}
func (f *FormatterText) Format(outputs Outputs) []byte {
	sort.Slice(outputs, func(i, j int) bool { return outputs[i].Timestamp.Before(outputs[j].Timestamp) })

	texts := []string{}
	for _, output := range outputs {
		text := fmt.Sprintf(
			"[%s] [%s] %s",
			output.Timestamp.Format("2006/01/02 15:04:05"),
			output.Username,
			output.Text,
		)
		for _, tfile := range output.TempFiles {
			text += fmt.Sprintf("\n(file: %s)", f.WriteFileName(tfile))
		}
		texts = append(texts, text)

		// replies
		sort.Slice(output.Replies, func(i, j int) bool { return output.Replies[i].Timestamp.Before(output.Replies[j].Timestamp) })
		for _, reply := range output.Replies {
			text := fmt.Sprintf(
				" | [%s] [%s] %s",
				reply.Timestamp.Format("2006/01/02 15:04:05"),
				reply.Username,
				strings.ReplaceAll(reply.Text, "\n", "\n | "),
			)
			for _, tfile := range reply.TempFiles {
				text += fmt.Sprintf("\n | (file: %s)", f.WriteFileName(tfile))
			}
			texts = append(texts, text)
		}
	}
	return []byte(strings.Join(texts, "\n"))
}
