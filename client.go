package drudge

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/yhat/scrape"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

const (
	URL = "http://www.drudgereport.com"
)

// Client is used for getting data from Drudge Report. It is safe for
// concurrent use. The zero value of client is ready to use.
//
// A Client caches the parsed HTML data that is fetched from Drudge
// Report. This cache is refreshed if at least an hour has passed
// since the last update to the cache.
type Client struct {
	// Client is the http.Client to use for fetching data.
	Client http.Client

	cache atomic.Value
}

// nodeCache holds both an html.Node and a time.Time. That's about it,
// actually.
type nodeCache struct {
	val *html.Node
	ts  time.Time
}

// cached returns the currently cached value. If there is no such
// value or the value has expired, it returns nil.
func (c *Client) cached() *html.Node {
	const maxAge = time.Hour

	cache, ok := c.cache.Load().(nodeCache)
	if !ok || (cache.val == nil) {
		return nil
	}

	if time.Since(cache.ts) > maxAge {
		c.cache.Store(nodeCache{})
		return nil
	}

	return cache.val
}

// page fetches and parses the main page of Drudge Report. It uses the
// Client's cache if possible, and caches the newly parsed node if no
// node is currently cached.
func (c *Client) page() (*html.Node, error) {
	if cached := c.cached(); cached != nil {
		return cached, nil
	}

	rsp, err := c.Client.Get(URL)
	if err != nil {
		return nil, err
	}
	defer rsp.Body.Close()

	node, err := html.Parse(rsp.Body)
	if err != nil {
		return nil, err
	}

	c.cache.Store(nodeCache{val: node, ts: time.Now()})
	return node, nil
}

// collect parses all of the Articles out of a node that it can find.
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

// get returns all of the articles in a specific section of the page.
// These sections are deliniated by tag IDs.
func (c *Client) get(section string) ([]Article, error) {
	node, err := c.page()
	if err != nil {
		return nil, err
	}

	node, ok := scrape.Find(node, scrape.ById(section))
	if !ok {
		return nil, errors.New("Couldn't find the top stories.")
	}

	return c.collect(node)
}

// Top returns the articles in the top section of the page. This
// includes the main headline.
func (c *Client) Top() (articles []Article, err error) {
	return c.get("app_topstories")
}

// Column returns the articles in one of the columns. The function
// panics if num is not in the range [1, 3].
func (c *Client) Column(num int) ([]Article, error) {
	if (num < 1) || (num > 3) {
		panic(fmt.Errorf("Bad column number: %v", num))
	}

	return c.get("app_col" + strconv.FormatInt(int64(num), 10))
}

// Article holds information about an article.
type Article struct {
	// Headline is the text used for the link to the article on Drudge
	// Report. It is likely not the article's actual headline.
	Headline string

	// URL is the URL of the article as linked to by Drudge Report.
	URL *url.URL

	// Image is the image associated with the article, if there is one.
	Image *url.URL
}
