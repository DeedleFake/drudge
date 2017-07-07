package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/DeedleFake/drudge"
)

var (
	client = &drudge.Client{
		Client: http.Client{
			Timeout: 30 * time.Second,
		},
	}

	sections = map[string]Section{
		"top": Section{
			Title: "Top",
			Fetch: client.Top,
		},
	}
)

func init() {
	fetch := func(i drudge.Column) func() ([]drudge.Article, error) {
		return func() ([]drudge.Article, error) {
			return client.Column(i)
		}
	}

	for i := 1; i <= 3; i++ {
		str := strconv.FormatInt(int64(i), 10)
		sections[str] = Section{
			Title: "Column " + str,
			Fetch: fetch(drudge.Column(i)),
		}
	}
}

type Section struct {
	Title string
	Fetch func() ([]drudge.Article, error)
}

func (s Section) Print(w io.Writer) error {
	articles, err := s.Fetch()
	if err != nil {
		return fmt.Errorf("Failed to fetch articles: %v", err)
	}

	ew := &errWriter{w: w}

	fmt.Fprintf(ew, "### %v ###\n\n", s.Title)
	for _, a := range articles {
		fmt.Fprintf(ew, "%v\n", a.Headline)
		fmt.Fprintf(ew, "\t%v\n", a.URL)
	}
	fmt.Fprintln(ew)

	return ew.err
}

type CSFlag []string

func (c CSFlag) String() string {
	return strings.Join(c, ",")
}

func (c *CSFlag) Set(val string) error {
	*c = strings.Split(val, ",")
	for i := range *c {
		(*c)[i] = strings.TrimSpace((*c)[i])
	}

	return nil
}

func main() {
	var flags struct {
		sec CSFlag
	}
	flags.sec = CSFlag{"top", "1", "2", "3"}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %v [options]\n\n", os.Args[0])

		fmt.Fprintln(os.Stderr, "Options:")
		flag.PrintDefaults()
	}
	flag.Var(&flags.sec, "sec", "Print `sections` in order given.")
	flag.Parse()

	for _, s := range flags.sec {
		sec, ok := sections[s]
		if !ok {
			fmt.Fprintf(os.Stderr, "Error: Unknown section: %q\n", s)
			os.Exit(1)
		}

		err := sec.Print(os.Stdout)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to print section %q: %v\n", s, err)
			os.Exit(1)
		}
	}
}

type errWriter struct {
	w   io.Writer
	err error
}

func (w *errWriter) Write(data []byte) (n int, err error) {
	if w.err != nil {
		return 0, w.err
	}

	n, err = w.w.Write(data)
	if err != nil {
		w.err = err
	}
	return n, err
}
