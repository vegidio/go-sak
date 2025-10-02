package main

import (
	"fmt"

	"github.com/vegidio/go-sak/fetch"
)

func main() {
	cookies := fetch.GetBrowserCookies("simpcity.cr")
	cookiesHeader := map[string]string{
		"Cookie": fetch.CookiesToHeader(cookies),
	}

	fmt.Println(cookiesHeader)

	f := fetch.New(cookiesHeader, 0)
	html, err := f.GetText("https://simpcity.cr/threads/jessica-nigri.9946/")

	if err != nil {
		fmt.Printf("Error fetching html: %v\n", err)
		return
	}

	fmt.Println(html)
}
