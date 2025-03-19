package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/go-shiori/go-readability"
	"github.com/schollz/progressbar/v3"
	epublifier "github.com/tjgurwara99/go-epublifier"
	"golang.org/x/net/html"
)

func renderBodyOnly(w io.Writer, node *html.Node, titleNode *html.Node) error {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			if child.Data == "body" {
				child.InsertBefore(titleNode, child.FirstChild)
				err := html.Render(w, child)
				return err
			}
		}
		renderBodyOnly(w, child, titleNode)
	}
	return nil
}

func removeDialogNodes(node *html.Node) {
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "dialog" {
			nextSibling := child.NextSibling
			child.Parent.RemoveChild(child)
			nextSibling.Parent.RemoveChild(nextSibling)
		}
		removeDialogNodes(child)
	}
}

func main() {
	n := 1199
	bar := progressbar.New(400)
	iterator := func() string {
		n++
		if n > 1599 {
			return ""
		}
		bar.Add(1)
		<-time.After(5 * time.Second)
		return fmt.Sprintf("https://www.lightnovelworld.com/novel/shadow-slave-1365/chapter-%d", n)
	}
	re := regexp.MustCompile(`[\s]*<p>[\s]*<span>[\s]*Tap the screen to use reading tools[\s]*<\/span>[\s]*<span>Tip: You can use left and right keyboard keys to browse between chapters\.<\/span>[\s]*<\/p>`)

	epub := epublifier.Epublifier{
		RequiresJS:     true,
		Title:          "Shadow Slave",
		Author:         "Guilty Three",
		URLIterator:    iterator,
		SavePath:       "shadow-slave-4.epub",
		CoverImagePath: "./nephis.webp",
		ChapterSanitiser: func(chapter readability.Article) (readability.Article, error) {
			node, err := html.Parse(strings.NewReader(chapter.Content))
			if err != nil {
				return chapter, fmt.Errorf("failed to parse html: %w", err)
			}
			buf := bytes.Buffer{}
			removeDialogNodes(node)
			err = renderBodyOnly(&buf, node, &html.Node{
				Type: html.ElementNode,
				Data: "h1",
				FirstChild: &html.Node{
					Type: html.TextNode,
					Data: chapter.Title,
				},
			})
			if err != nil {
				return chapter, fmt.Errorf("failed to sanitise chapter: %w", err)
			}
			chapter.Content = buf.String()
			chapter.Content = re.ReplaceAllString(chapter.Content, "")
			return chapter, nil
		},
	}
	if err := epub.Create(); err != nil {
		log.Fatal(err)
	}
}
