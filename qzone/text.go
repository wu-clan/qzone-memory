package qzone

import (
	"html"
	"regexp"
	"strings"
)

var htmlTagPattern = regexp.MustCompile(`(?s)<[^>]*>`)

func cleanPlainText(value string) string {
	value = html.UnescapeString(value)
	value = strings.ReplaceAll(value, "\u00a0", " ")
	value = strings.ReplaceAll(value, "\r", "")
	value = strings.ReplaceAll(value, "\n", " ")
	if strings.Contains(value, "<") && strings.Contains(value, ">") {
		value = htmlTagPattern.ReplaceAllString(value, " ")
	}
	value = strings.Join(strings.Fields(value), " ")
	return strings.TrimSpace(value)
}

func looksLikeSystemAlbum(name, description string) bool {
	name = cleanPlainText(name)
	description = strings.ToLower(description)

	if name == "相册回溯记录" {
		return true
	}
	if strings.Contains(description, `class="f-single`) || strings.Contains(description, "feed_") {
		return true
	}
	return false
}
