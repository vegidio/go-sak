package fetch

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all"
)

// httpOnlyPrefix marks an HttpOnly cookie in the files curl, yt-dlp and the browser export extensions produce. The
// line is a normal record behind it, and it is usually the session cookie the caller actually wants.
const httpOnlyPrefix = "#HttpOnly_"

// maxCookieLine is the scanner's buffer ceiling. bufio's own default of 64 KiB is small enough that a single JWT or
// session blob can exceed it.
const maxCookieLine = 1024 * 1024

// GetFileCookies reads cookies from a file in the Netscape cookie-jar format.
//
// Lines that are blank, commented, or do not carry the seven tab-separated fields the format specifies are skipped.
// An "#HttpOnly_" prefix is stripped rather than treated as a comment.
//
// # Parameters:
//   - filePath: path to the cookie file
//
// # Returns:
//   - the cookies found in the file
//   - an error if the file cannot be opened or cannot be read to the end
func GetFileCookies(filePath string) ([]Cookie, error) {
	cookies := make([]Cookie, 0)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), maxCookieLine)

	for scanner.Scan() {
		line := scanner.Text()

		// An HttpOnly record is a real cookie wearing a comment prefix, not a comment.
		line = strings.TrimPrefix(line, httpOnlyPrefix)

		// Skip comments & blank lines
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		// Spec says 7 TAB-separated fields
		parts := strings.Split(line, "\t")
		if len(parts) != 7 {
			continue
		}

		name := parts[5]
		value := parts[6]

		cookies = append(cookies, Cookie{
			Name:  name,
			Value: value,
		})
	}

	// Without this, an over-long line or a read error ends the loop and returns a silently truncated cookie list
	// with a nil error, leaving the caller to authenticate with a partial jar.
	if err = scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read cookie file %s: %w", filePath, err)
	}

	return cookies, nil
}

func GetBrowserCookies(domain string) []Cookie {
	cookies := make([]Cookie, 0)

	cookiesSeq := kooky.TraverseCookies(context.Background(), kooky.Valid, kooky.DomainHasSuffix(domain)).OnlyCookies()
	for cookie := range cookiesSeq {
		cookies = append(cookies, Cookie{
			Name:  cookie.Name,
			Value: cookie.Value,
		})
	}

	return cookies
}

func CookiesToHeader(cookies []Cookie) string {
	var sb strings.Builder
	for i, cookie := range cookies {
		if i > 0 {
			sb.WriteString("; ")
		}

		sb.WriteString(cookie.Name)
		sb.WriteByte('=')
		sb.WriteString(cookie.Value)
	}

	return sb.String()
}
