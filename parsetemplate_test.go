package main

import (
	"github.com/stretchr/testify/require"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTemplate(t *testing.T) {
	msgHolder := &MsgHolder{
		strings: map[string][]TranslationString{},
	}

	cwd, _ := os.Getwd()
	basePath := filepath.Join(cwd, "testdata")

	err := filepath.Walk(filepath.Join(basePath, "templates"), func(path string, info os.FileInfo, err error) error {
		if !info.Mode().IsRegular() {
			return nil
		}
		return parseTemplate(path, msgHolder)
	})
	require.Nil(t, err)

	defaultDom, ok := msgHolder.strings["default"]
	require.True(t, ok, "Expected default domain to be present")

	messageSingular := map[string]bool{}
	messagePlural := map[string]bool{}
	for _, msg := range defaultDom {
		var prefix string
		if msg.Context != "" {
			prefix = msg.Context + ":"
		}
		messageSingular[prefix+msg.Singular] = true

		if msg.Plural != "" {
			messagePlural[prefix+msg.Plural] = true
		}
	}

	expectedSingular := []string{
		"String from trans",
		"String from trans to var",
		"testctx:String from trans with context",
		"\nString from blocktrans\n",
		"\nString from blocktrans with plural\n",
	}

	expectedPlural := []string{
		"\nPlural for blocktrans\n",
	}

	for _, expected := range expectedSingular {
		require.True(t, messageSingular[expected], "expected to find string '%s' in messages", expected)
	}

	for _, expected := range expectedPlural {
		require.True(t, messagePlural[expected], "expected to find string '%s' in plural messages", expected)
	}
}
