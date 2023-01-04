package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseGo(t *testing.T) {
	msgHolder := &MsgHolder{
		strings: map[string][]TranslationString{},
	}

	cwd, _ := os.Getwd()
	basePath := filepath.Join(cwd, "testdata")
	err := parseGo(basePath, []string{"."}, msgHolder)
	if err != nil {
		t.Fatalf("parseGo returned error: %v", err)
		return
	}

	defaultDom, ok := msgHolder.strings["default"]
	require.True(t, ok, "Expected default domain to be present")

	messageMap := map[string]bool{}
	for _, msg := range defaultDom {
		messageMap[msg.Singular] = true
	}

	require.True(t, messageMap["String from gotext package"], "expected to find string in messages")
	require.True(t, messageMap["String from gotext.Locale"], "expected to find string in messages")
	require.True(t, messageMap["String from gotext.Po"], "expected to find string in messages")
	require.True(t, messageMap["String from gotext.Mo"], "expected to find string in messages")
}
