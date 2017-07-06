package drudge

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/yhat/scrape"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	URL = "http://www.drudgereport.com"
)

type Client struct {
	Client http.Client
}

func (c *Client) page() (*html.Node, error) {
	rsp, err := c.Client.Get(URL)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	return html.Parse(rsp.Body)
}

func (c *Client) collect(node *html.Node) (articles []Article, err error) {
	images := make(map[string]struct{})

	links := scrape.FindAll(node, scrape.ByTag(atom.A))
	for _, link := range links {
		href, err := url.Parse(scrape.Attr(link, "href"))
		if err != nil {
			return nil, err
		}

		article := Article{
			Headline: scrape.Text(link),
			URL:      href,
		}

		img, ok := scrape.FindPrevSibling(link, scrape.ByTag(atom.Img))
		if ok {
			src := scrape.Attr(img, "src")
			if _, ok := images[src]; !ok {
				images[src] = struct{}{}

				src, err := url.Parse(src)
				if err != nil {
					return nil, err
				}
				article.Image = src
			}
		}

		articles = append(articles, article)
	}

	return articles, nil
}

func (c *Client) Top() (articles []Article, err error) {
	node, err := c.page()
	if err != nil {
		return nil, err
	}

	node, ok := scrape.Find(node, scrape.ById("app_topstories"))
	if !ok {
		return nil, errors.New("Couldn't find the top stories.")
	}

	return c.collect(node)
}

type Article struct {
	Headline string
	URL      *url.URL
	Image    *url.URL
}
