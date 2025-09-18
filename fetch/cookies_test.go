package fetch

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFileCookies(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Create a temporary file with valid cookie data
		content := `# Netscape HTTP Cookie File
.example.com	TRUE	/	FALSE	1234567890	sessionid	abc123
.example.com	TRUE	/api	TRUE	1234567891	authtoken	xyz789
.google.com	FALSE	/search	FALSE	0	preferences	theme=dark`

		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		cookies, err := GetFileCookies(tempFile)

		require.NoError(t, err)
		assert.Len(t, cookies, 3)

		expectedCookies := []Cookie{
			{Name: "sessionid", Value: "abc123"},
			{Name: "authtoken", Value: "xyz789"},
			{Name: "preferences", Value: "theme=dark"},
		}

		assert.Equal(t, expectedCookies, cookies)
	})

	t.Run("EmptyFile", func(t *testing.T) {
		tempFile := createTempFile(t, "")
		defer os.Remove(tempFile)

		cookies, err := GetFileCookies(tempFile)

		require.NoError(t, err)
		assert.Empty(t, cookies)
	})

	t.Run("OnlyCommentsAndBlankLines", func(t *testing.T) {
		content := `# Netscape HTTP Cookie File
# This is a comment

# Another comment
`

		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		cookies, err := GetFileCookies(tempFile)

		require.NoError(t, err)
		assert.Empty(t, cookies)
	})

	t.Run("IgnoreInvalidLines", func(t *testing.T) {
		content := `.example.com	TRUE	/	FALSE	1234567890	sessionid	abc123
invalid line with only 3	fields	here
.example.com	TRUE	/api	TRUE	1234567891	authtoken	xyz789
another invalid line
.google.com	FALSE	/search	FALSE	0	preferences	theme=dark`

		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		cookies, err := GetFileCookies(tempFile)

		require.NoError(t, err)
		assert.Len(t, cookies, 3)

		expectedCookies := []Cookie{
			{Name: "sessionid", Value: "abc123"},
			{Name: "authtoken", Value: "xyz789"},
			{Name: "preferences", Value: "theme=dark"},
		}

		assert.Equal(t, expectedCookies, cookies)
	})

	t.Run("MixedValidAndInvalidLines", func(t *testing.T) {
		content := `# Header comment
.example.com	TRUE	/	FALSE	1234567890	sessionid	abc123
# Another comment
invalid	line
.google.com	FALSE	/search	FALSE	0	preferences	theme=dark

# Empty line above and comment
.test.com	TRUE	/	TRUE	1234567892	testcookie	testvalue`

		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		cookies, err := GetFileCookies(tempFile)

		require.NoError(t, err)
		assert.Len(t, cookies, 3)

		expectedCookies := []Cookie{
			{Name: "sessionid", Value: "abc123"},
			{Name: "preferences", Value: "theme=dark"},
			{Name: "testcookie", Value: "testvalue"},
		}

		assert.Equal(t, expectedCookies, cookies)
	})

	t.Run("EmptyNameAndValue", func(t *testing.T) {
		content := `.example.com	TRUE	/	FALSE	1234567890		
.google.com	FALSE	/search	FALSE	0	emptycookie	
.test.com	TRUE	/	TRUE	1234567892	normalcookie	normalvalue`

		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		cookies, err := GetFileCookies(tempFile)

		require.NoError(t, err)
		assert.Len(t, cookies, 3)

		expectedCookies := []Cookie{
			{Name: "", Value: ""},
			{Name: "emptycookie", Value: ""},
			{Name: "normalcookie", Value: "normalvalue"},
		}

		assert.Equal(t, expectedCookies, cookies)
	})

	t.Run("SpecialCharactersInCookies", func(t *testing.T) {
		content := `.example.com	TRUE	/	FALSE	1234567890	special_cookie-1	value with spaces
.example.com	TRUE	/api	TRUE	1234567891	cookie.with.dots	value=with=equals
.google.com	FALSE	/search	FALSE	0	cookie-with-dashes	value;with;semicolons`

		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		cookies, err := GetFileCookies(tempFile)

		require.NoError(t, err)
		assert.Len(t, cookies, 3)

		expectedCookies := []Cookie{
			{Name: "special_cookie-1", Value: "value with spaces"},
			{Name: "cookie.with.dots", Value: "value=with=equals"},
			{Name: "cookie-with-dashes", Value: "value;with;semicolons"},
		}

		assert.Equal(t, expectedCookies, cookies)
	})

	t.Run("FileNotFound", func(t *testing.T) {
		cookies, err := GetFileCookies("/path/to/nonexistent/file.txt")

		assert.Error(t, err)
		assert.Nil(t, cookies)
		assert.Contains(t, err.Error(), "no such file or directory")
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test when running as root")
		}

		// Create a file without read permissions
		tempFile := createTempFile(t, "test content")
		defer os.Remove(tempFile)

		err := os.Chmod(tempFile, 0000)
		require.NoError(t, err)

		cookies, err := GetFileCookies(tempFile)

		assert.Error(t, err)
		assert.Nil(t, cookies)
		assert.Contains(t, err.Error(), "permission denied")
	})

	t.Run("WindowsLineEndings", func(t *testing.T) {
		content := ".example.com\tTRUE\t/\tFALSE\t1234567890\tsessionid\tabc123\r\n" +
			".google.com\tFALSE\t/search\tFALSE\t0\tpreferences\ttheme=dark\r\n"

		tempFile := createTempFile(t, content)
		defer os.Remove(tempFile)

		cookies, err := GetFileCookies(tempFile)

		require.NoError(t, err)
		assert.Len(t, cookies, 2)

		expectedCookies := []Cookie{
			{Name: "sessionid", Value: "abc123"},
			{Name: "preferences", Value: "theme=dark"},
		}

		assert.Equal(t, expectedCookies, cookies)
	})
}

func TestCookiesToHeader(t *testing.T) {
	t.Run("EmptySlice", func(t *testing.T) {
		cookies := []Cookie{}
		result := CookiesToHeader(cookies)

		assert.Equal(t, "", result)
	})

	t.Run("SingleCookie", func(t *testing.T) {
		cookies := []Cookie{
			{Name: "sessionid", Value: "abc123"},
		}
		result := CookiesToHeader(cookies)

		assert.Equal(t, "sessionid=abc123", result)
	})

	t.Run("MultipleCookies", func(t *testing.T) {
		cookies := []Cookie{
			{Name: "sessionid", Value: "abc123"},
			{Name: "authtoken", Value: "xyz789"},
			{Name: "preferences", Value: "theme=dark"},
		}
		result := CookiesToHeader(cookies)

		assert.Equal(t, "sessionid=abc123; authtoken=xyz789; preferences=theme=dark", result)
	})

	t.Run("CookiesWithEmptyValues", func(t *testing.T) {
		cookies := []Cookie{
			{Name: "sessionid", Value: "abc123"},
			{Name: "emptycookie", Value: ""},
			{Name: "preferences", Value: "theme=dark"},
		}
		result := CookiesToHeader(cookies)

		assert.Equal(t, "sessionid=abc123; emptycookie=; preferences=theme=dark", result)
	})

	t.Run("CookiesWithEmptyNames", func(t *testing.T) {
		cookies := []Cookie{
			{Name: "sessionid", Value: "abc123"},
			{Name: "", Value: "orphanvalue"},
			{Name: "preferences", Value: "theme=dark"},
		}
		result := CookiesToHeader(cookies)

		assert.Equal(t, "sessionid=abc123; =orphanvalue; preferences=theme=dark", result)
	})

	t.Run("CookiesWithSpecialCharacters", func(t *testing.T) {
		cookies := []Cookie{
			{Name: "special_cookie-1", Value: "value with spaces"},
			{Name: "cookie.with.dots", Value: "value=with=equals"},
			{Name: "cookie-with-dashes", Value: "value;with;semicolons"},
		}
		result := CookiesToHeader(cookies)

		expected := "special_cookie-1=value with spaces; cookie.with.dots=value=with=equals; cookie-with-dashes=value;with;semicolons"
		assert.Equal(t, expected, result)
	})

	t.Run("CookiesWithSpacesInNames", func(t *testing.T) {
		cookies := []Cookie{
			{Name: "cookie with spaces", Value: "value1"},
			{Name: "normal", Value: "value2"},
		}
		result := CookiesToHeader(cookies)

		assert.Equal(t, "cookie with spaces=value1; normal=value2", result)
	})

	t.Run("CookiesWithUnicodeCharacters", func(t *testing.T) {
		cookies := []Cookie{
			{Name: "unicode_cookie", Value: "café"},
			{Name: "emoji_cookie", Value: "🍪"},
			{Name: "chinese", Value: "测试"},
		}
		result := CookiesToHeader(cookies)

		expected := "unicode_cookie=café; emoji_cookie=🍪; chinese=测试"
		assert.Equal(t, expected, result)
	})
}

func TestCookie(t *testing.T) {
	t.Run("CookieStructFields", func(t *testing.T) {
		cookie := Cookie{
			Name:  "testcookie",
			Value: "testvalue",
		}

		assert.Equal(t, "testcookie", cookie.Name)
		assert.Equal(t, "testvalue", cookie.Value)
	})

	t.Run("EmptyCookie", func(t *testing.T) {
		cookie := Cookie{}

		assert.Equal(t, "", cookie.Name)
		assert.Equal(t, "", cookie.Value)
	})

	t.Run("CookieComparison", func(t *testing.T) {
		cookie1 := Cookie{Name: "test", Value: "value"}
		cookie2 := Cookie{Name: "test", Value: "value"}
		cookie3 := Cookie{Name: "different", Value: "value"}

		assert.Equal(t, cookie1, cookie2)
		assert.NotEqual(t, cookie1, cookie3)
	})
}

// Helper function to create temporary files for testing
func createTempFile(t *testing.T, content string) string {
	t.Helper()

	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "cookies.txt")

	err := os.WriteFile(tempFile, []byte(content), 0644)
	require.NoError(t, err)

	return tempFile
}
