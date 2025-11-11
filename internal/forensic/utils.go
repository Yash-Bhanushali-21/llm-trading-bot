package forensic

import "strings"

// containsAny checks if the text contains any of the given keywords
func containsAny(text string, keywords []string) bool {
	textLower := strings.ToLower(text)
	for _, keyword := range keywords {
		if strings.Contains(textLower, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}
