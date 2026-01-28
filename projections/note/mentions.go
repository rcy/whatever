package note

import (
	"regexp"
	"strings"
)

var mentionRe = regexp.MustCompile(`(?i)(?:^|[^a-z0-9_])@([a-z0-9_]+)`)

func extractMentions(text string) []string {
	matches := mentionRe.FindAllStringSubmatch(text, -1)

	seen := make(map[string]struct{})
	var result []string

	for _, m := range matches {
		handle := strings.ToLower(m[1]) // normalize
		if _, ok := seen[handle]; ok {
			continue
		}
		seen[handle] = struct{}{}
		result = append(result, handle)
	}

	return result
}
