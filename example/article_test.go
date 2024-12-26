package main_test

import (
	"log"

	"github.com/tjgurwara99/go-epublifier"
)

func Example() {
	n := 0
	iterator := func() string {
		n++
		if n > 1 {
			return ""
		}
		return "https://www.theguardian.com/us-news/2023/mar/31/trump-appear-court-new-york-tuesday-criminal-charges"
	}
	epub := epublifier.Epublifier{
		Title:       "Trump to appear in New York court on Tuesday to answer criminal charges",
		Author:      "Joan E Greve",
		URLIterator: iterator,
		SavePath:    "donald-trump-article.epub",
	}
	if err := epub.Create(); err != nil {
		log.Fatal(err)
	}
	// Output:
}

func WithJS() {} // need this for linters to not complain with the example

func ExampleWithJS() {
	n := 0
	iterator := func() string {
		n++
		if n > 1 {
			return ""
		}
		return "https://www.theguardian.com/us-news/2023/mar/31/trump-appear-court-new-york-tuesday-criminal-charges"
	}
	epub := epublifier.Epublifier{
		Title:       "Trump to appear in New York court on Tuesday to answer criminal charges",
		Author:      "Joan E Greve",
		URLIterator: iterator,
		SavePath:    "donald-trump-article.epub",
		RequiresJS:  true, // use a headless browser to load the pages
	}
	if err := epub.Create(); err != nil {
		log.Fatal(err)
	}
	// Output:
}
