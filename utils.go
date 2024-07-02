package archive

import (
	"encoding/base64"
	"fmt"
	"os"
)

func firstString(slice []string) string {
	for _, s := range slice {
		if s != "" {
			return s
		}
	}
	return ""
}

// NOTE: os.Getenv(ENVNAME) or os.Getenv(ENVNAME_BASE64)
func Getenv(env string) string {
	envPrefix := "SA_"
	plain := os.Getenv(envPrefix + env)
	b64 := os.Getenv(envPrefix + env + "_BASE64")

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
