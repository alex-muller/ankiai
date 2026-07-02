package lib

import "strings"

func CleanWord(text string) string {
	text = strings.ToLower(text)
	// text = strings.ReplaceAll(text, "-", " ")
	suffixes := []string{"'s", "’s", "'", "’"}
	words := strings.Fields(text)

	for i, word := range words {
		for _, suffix := range suffixes {
			if strings.HasSuffix(word, suffix) {
				words[i] = strings.TrimSuffix(word, suffix)

				break
			}
		}
	}

	return strings.Join(words, " ")
}

func CleanPhraseAndSplit(text string) []string {
	var out []string
	words := strings.Split(text, " ")
	for _, word := range words {
		out = append(out, CleanWord(word))
	}
	return out
}

func CleanWords(text []string) []string {
	var out []string
	for _, word := range text {
		out = append(out, CleanWord(word))
	}
	return out
}
