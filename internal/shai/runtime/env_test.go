package shai

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test #87: Empty environment values are parsed correctly
func TestEnv_EmptyValuesHandled(t *testing.T) {
	// Set actual environment variables
	os.Setenv("TEST_EMPTY_KEY", "")
	os.Setenv("TEST_KEY_WITH_VALUE", "value1")
	defer func() {
		os.Unsetenv("TEST_EMPTY_KEY")
		os.Unsetenv("TEST_KEY_WITH_VALUE")
	}()

	result := hostEnvMap()

	// Verify empty key exists and has empty value
	val, exists := result["TEST_EMPTY_KEY"]
	assert.True(t, exists, "TEST_EMPTY_KEY should exist in map")
	assert.Equal(t, "", val, "TEST_EMPTY_KEY should have empty string value")

	// Verify key with value works correctly
	assert.Equal(t, "value1", result["TEST_KEY_WITH_VALUE"], "TEST_KEY_WITH_VALUE should have value1")
}

// Test #88: Special characters in environment values preserved
func TestEnv_SpecialCharactersPreserved(t *testing.T) {
	// Set environment variables with special characters
	testCases := map[string]string{
		"TEST_QUOTES_DOUBLE":  "\"quoted value\"",
		"TEST_QUOTES_SINGLE":  "'single quoted'",
		"TEST_NEWLINE":        "line1\\nline2",
		"TEST_SPACES":         "  leading and trailing  ",
		"TEST_SHELL_CHARS":    "foo && bar || baz",
		"TEST_PATH_LIKE":      "/usr/bin:/usr/local/bin",
		"TEST_EQUALS_IN_VALUE":"key=value=another",
		"TEST_SPECIAL":        "!@#$%^&*()",
		"TEST_UNICODE":        "héllo wörld 你好",
	}

	for key, value := range testCases {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range testCases {
			os.Unsetenv(key)
		}
	}()

	result := hostEnvMap()

	// Verify all special characters are preserved exactly
	for key, expectedValue := range testCases {
		actualValue, exists := result[key]
		assert.True(t, exists, "%s should exist", key)
		assert.Equal(t, expectedValue, actualValue, "%s value should be preserved", key)
	}
}

// Additional test: Very long environment variable values
func TestEnv_VeryLongValues(t *testing.T) {
	// Create a very long value (10KB)
	longValue := ""
	for i := 0; i < 10240; i++ {
		longValue += "a"
	}

	os.Setenv("TEST_LONG_KEY", longValue)
	defer os.Unsetenv("TEST_LONG_KEY")

	result := hostEnvMap()

	actualValue, exists := result["TEST_LONG_KEY"]
	assert.True(t, exists, "Long key should exist")
	assert.Equal(t, longValue, actualValue, "Long value should be preserved without truncation")
	assert.Len(t, actualValue, 10240, "Long value should maintain full length")
}

// Additional test: Unicode in values
func TestEnv_UnicodeHandling(t *testing.T) {
	testCases := map[string]string{
		"TEST_ENGLISH": "hello world",
		"TEST_SPANISH": "héllo mündø",
		"TEST_CHINESE": "你好世界",
		"TEST_EMOJI":   "🚀🎉✨",
		"TEST_MIXED":   "hello 世界 🌍",
	}

	for key, value := range testCases {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range testCases {
			os.Unsetenv(key)
		}
	}()

	result := hostEnvMap()

	for key, expectedValue := range testCases {
		assert.Equal(t, expectedValue, result[key], "%s should preserve Unicode", key)
	}
}

// Additional test: First equals sign determines key/value split
func TestEnv_FirstEqualsSplitsKeyValue(t *testing.T) {
	testCases := map[string]string{
		"TEST_URL":      "https://example.com/path?foo=bar&baz=qux",
		"TEST_EQUATION": "x=y+z",
		"TEST_JSON":     "{\"key\":\"value\",\"foo\":\"bar\"}",
	}

	for key, value := range testCases {
		os.Setenv(key, value)
	}
	defer func() {
		for key := range testCases {
			os.Unsetenv(key)
		}
	}()

	result := hostEnvMap()

	for key, expectedValue := range testCases {
		assert.Equal(t, expectedValue, result[key], "%s should preserve equals signs in value", key)
	}
}

// Test: hostUserIDs returns non-zero UIDs
func TestHostUserIDs_ReturnsNonZeroUIDs(t *testing.T) {
	uid, gid := hostUserIDs()

	// UIDs should never be "0" - fallback to "4747" if host is root
	assert.NotEqual(t, "0", uid, "UID should never be 0 (uses fallback 4747)")
	assert.NotEqual(t, "0", gid, "GID should never be 0 (uses fallback 4747)")
	assert.NotEqual(t, "", uid, "UID should not be empty")
	assert.NotEqual(t, "", gid, "GID should not be empty")

	// Should be valid numeric strings
	assert.Regexp(t, `^\d+$`, uid, "UID should be numeric")
	assert.Regexp(t, `^\d+$`, gid, "GID should be numeric")
}
