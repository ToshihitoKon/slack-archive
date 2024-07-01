package archive

func firstString(slice []string) string {
	for _, s := range slice {
		if s != "" {
			return s
		}
	}
	return ""
}
