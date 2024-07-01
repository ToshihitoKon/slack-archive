package archive

import (
	"encoding/base64"
	"fmt"
	"os"
)

// NOTE: os.Getenv(ENVNAME) or os.Getenv(ENVNAME_BASE64)
func getEnv(env string) string {
	plain := os.Getenv(env)
	b64 := os.Getenv(env + "_BASE64")
	if plain != "" {
		return plain
	}
	if b64 != "" {
		decoded, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			panic(fmt.Errorf("error: Environment variable decode: %s: %w", env+"_BASE64", err))
		}
		return string(decoded)
	}
	return ""
}

func firstString(slice []string) string {
	for _, s := range slice {
		if s != "" {
			return s
		}
	}
	return ""
}
