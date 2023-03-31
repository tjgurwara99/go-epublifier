package epublifier

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/bmaupin/go-epub"
	"github.com/go-shiori/go-readability"
	"golang.org/x/net/html"
)

type Epublifier struct {
	Title             string
	Author            string
	CoverImagePath    string
	CoverImageCSSPath string
	URLIterator       func() string
	ChapterSanitiser  func(readability.Article) (readability.Article, error)
	SavePath          string
}

type EpublifierError string

func (e EpublifierError) Error() string {
	return string(e)
}

const (
	TitleNotProvidedErr             EpublifierError = "title not provided"
	IteratorNotProvidedErr          EpublifierError = "iterator not provided to iterate through urls"
	AuthorNotProvidedErr            EpublifierError = "author not provided"
	SavePathNotProvidedErr          EpublifierError = "save path not provided"
	CoverImageCSSPathNotProvidedErr EpublifierError = "cover image css path which is necessary when providing cover image"
)

func (e *Epublifier) validate() error {
	var errs []error
	if e.Title == "" {
		errs = append(errs, TitleNotProvidedErr)
	}
	if e.Author == "" {
		errs = append(errs, AuthorNotProvidedErr)
	}
	if e.SavePath == "" {
		errs = append(errs, SavePathNotProvidedErr)
	}
	if e.URLIterator == nil {
		errs = append(errs, IteratorNotProvidedErr)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

const defaultCoverCSSData = `body {
  background-color: #FFFFFF;
  margin-bottom: 0px;
  margin-left: 0px;
  margin-right: 0px;
  margin-top: 0px;
  text-align: center;
}

img {
  max-height: 100%;
  max-width: 100%;
}
`

func (e *Epublifier) setCoverImage(book *epub.Epub) (func(), error) {
	cleanup := func() {}
	if e.CoverImagePath == "" {
		return cleanup, nil
	}
	extension := path.Ext(e.CoverImagePath)
	coverImage, err := book.AddImage(e.CoverImagePath, "cover"+extension)
	if err != nil {
		return cleanup, nil
	}
	var file *os.File
	if e.CoverImageCSSPath == "" {
		file, err = os.CreateTemp("", "*.css")
		if err != nil {
			return cleanup, fmt.Errorf("failed to create temp file for default css: %w", err)
		}
		cleanup = func() { os.Remove(file.Name()) }
		_, err = file.Write([]byte(defaultCoverCSSData))
		if err != nil {
			return cleanup, fmt.Errorf("failed to write to temp css file: %w", err)
		}
		err = file.Close()
		if err != nil {
			return cleanup, fmt.Errorf("failed to close temporary css file: %w", err)
		}
		e.CoverImageCSSPath = file.Name()
	}
	coverCSS, err := book.AddCSS(e.CoverImageCSSPath, "cover.css")
	if err != nil {
		return cleanup, err
	}
	book.SetCover(coverImage, coverCSS)
	return cleanup, err
}

func onlyBody(w io.Writer, n *html.Node) error {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			if child.Data == "body" {
				err := html.Render(w, child)
				return err
			}
		}
		onlyBody(w, child)
	}
	return nil
}

func removeFirefoxReaderExtraChild(n *html.Node) {
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode {
			if child.Data == "section" {
				child.RemoveChild(child.LastChild)
			}
		}
		removeFirefoxReaderExtraChild(child)
	}
}

func DefaultChapterSanitiser(a readability.Article) (readability.Article, error) {
	node, err := html.Parse(strings.NewReader(a.Content))
	if err != nil {
		return a, fmt.Errorf("failed to parse html: %w", err)
	}
	removeFirefoxReaderExtraChild(node)
	buf := bytes.Buffer{}
	err = onlyBody(&buf, node)
	if err != nil {
		return a, fmt.Errorf("failed to sanitise readability article: %w", err)
	}
	a.Content = buf.String()
	return a, nil
}

func (e *Epublifier) Create() error {
	err := e.validate()
	if err != nil {
		return err
	}
	book := epub.NewEpub(e.Title)
	book.SetAuthor(e.Author)
	cleanupFunc, err := e.setCoverImage(book)
	defer cleanupFunc()
	if err != nil {
		return err
	}
	for url := e.URLIterator(); url != ""; url = e.URLIterator() {
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Fatal(err)
		}
		req.Header.Set("Accept", "text/html")
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/537.11 (KHTML, like Gecko) Chrome/23.0.1271.64 Safari/537.11")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		chapter, err := readability.FromReader(resp.Body, resp.Request.URL)
		if err != nil {
			return fmt.Errorf("failed to find readerview on the provided url: %w", err)
		}
		if e.ChapterSanitiser == nil {
			e.ChapterSanitiser = DefaultChapterSanitiser
		}
		chapter, err = e.ChapterSanitiser(chapter)
		if err != nil {
			return fmt.Errorf("failed to sanitise chapter: %w", err)
		}
		book.AddSection(chapter.Content, chapter.Title, strings.ToLower(strings.ReplaceAll(chapter.Title, " ", "-")), "")
	}
	return book.Write(e.SavePath)
}
