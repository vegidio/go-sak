package fetch

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/browserutils/kooky"
	_ "github.com/browserutils/kooky/browser/all"
	"github.com/samber/lo"
)

// Cookie represents a key-value pair for a typical HTTP cookie.
type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func GetFileCookies(filePath string) ([]Cookie, error) {
	cookies := make([]Cookie, 0)

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

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
	parts := lo.Map(cookies, func(cookie Cookie, index int) string {
		return fmt.Sprintf("%s=%s", cookie.Name, cookie.Value)
	})

	return strings.Join(parts, "; ")
}
