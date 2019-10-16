package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

//PreviewImage represents a preview image for a page
type PreviewImage struct {
	URL       string `json:"url,omitempty"`
	SecureURL string `json:"secureURL,omitempty"`
	Type      string `json:"type,omitempty"`
	Width     int    `json:"width,omitempty"`
	Height    int    `json:"height,omitempty"`
	Alt       string `json:"alt,omitempty"`
}

//PageSummary represents summary properties for a web page
type PageSummary struct {
	Type        string          `json:"type,omitempty"`
	URL         string          `json:"url,omitempty"`
	Title       string          `json:"title,omitempty"`
	SiteName    string          `json:"siteName,omitempty"`
	Description string          `json:"description,omitempty"`
	Author      string          `json:"author,omitempty"`
	Keywords    []string        `json:"keywords,omitempty"`
	Icon        *PreviewImage   `json:"icon,omitempty"`
	Images      []*PreviewImage `json:"images,omitempty"`
}

//SummaryHandler handles requests for the page summary API.
//This API expects one query string parameter named `url`,
//which should contain a URL to a web page. It responds with
//a JSON-encoded PageSummary struct containing the page summary
//meta-data.
func SummaryHandler(w http.ResponseWriter, r *http.Request) {

	w.Header().Set("Access-Control-Allow-Origin", "*")

	pageURL := r.URL.Query().Get("url")
	if len(pageURL) == 0 {
		http.Error(w, "No query found in the requested url", http.StatusBadRequest)
		return
	}

	resp, err := fetchHTML(pageURL)
	if err != nil {
		log.Fatalf("error fetching URL: %v\n", err)
	}

	PageSummary, err := extractSummary(pageURL, resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error extracting summary: %v", err), http.StatusBadRequest)
		return
	}

	defer resp.Close()

	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PageSummary)
}

//fetchHTML fetches `pageURL` and returns the body stream or an error.
//Errors are returned if the response status code is an error (>=400),
//or if the content type indicates the URL is not an HTML page.
func fetchHTML(pageURL string) (io.ReadCloser, error) {
	if !strings.HasPrefix(pageURL, "https://") && !strings.HasPrefix(pageURL, "http://") {
		pageURL = "https://" + pageURL
	}
	resp, err := http.Get(pageURL)

	if err != nil {
		log.Fatalf("error fetching URL: %v\n", err)
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("response status code was %d", resp.StatusCode)
	}

	ctype := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ctype, "text/html") {
		return nil, fmt.Errorf("response content type was %s, not text/html", ctype)
	}

	return resp.Body, err
}

//extractSummary tokenizes the `htmlStream` and populates a PageSummary
//struct with the page's summary meta-data.
func extractSummary(pageURL string, htmlStream io.ReadCloser) (*PageSummary, error) {

	summary := &PageSummary{}

	tokenizer := html.NewTokenizer(htmlStream)

	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				break
			}
			log.Fatalf("error tokenizing HTML: %v", tokenizer.Err())
		}

		token := tokenizer.Token()
		if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
			if token.Data == "meta" {
				prop, _ := getAttr(token, "property")
				name, _ := getAttr(token, "name")
				cont, _ := getAttr(token, "content")
				if prop == "og:type" {
					summary.Type = cont
				} else if prop == "og:url" {
					summary.URL = cont
				} else if prop == "og:title" {
					summary.Title = cont
				} else if prop == "og:site_name" {
					summary.SiteName = cont
				}

				if prop == "og:description" {
					summary.Description = cont
				} else if name == "description" && summary.Description == "" { // no open graph description
					summary.Description = cont
				}

				if name == "author" {
					summary.Author = cont
				}

				if name == "keywords" {
					if strings.IndexAny(cont, ",") != -1 { // remove all whitespace and split into slice
						summary.Keywords = strings.Split(strings.Replace(cont, " ", "", -1), ",")
					} else {
						summary.Keywords = []string{cont}
					}
				}

				if strings.HasPrefix(prop, "og:image") {
					if strings.HasPrefix(prop, "og:image:") { // is property of last PreviewImage made
						lastImage := summary.Images[len(summary.Images)-1]
						lastImage = buildImage(prop, pageURL, lastImage, cont)
					} else {
						summary.Images = append(summary.Images, buildImage(prop, pageURL, &PreviewImage{}, cont))
					}
				}
			}

			if token.Data == "title" && summary.Title == "" {
				temp := tokenizer.Next()
				if temp == html.TextToken {
					summary.Title = tokenizer.Token().Data
				}
			}

			if token.Data == "link" {
				summary.Icon = buildIcon(token, pageURL)
			}
		}

		if tokenType == html.EndTagToken && token.Data == "head" {
			break
		}

	}

	return summary, tokenizer.Err()
}

// getAttr takes in an html token from a tokenizer along with a desired attribute string
// and searches/returns the val of that attribute if found. Otherwise it returns an empty string
// as well as a custom error
func getAttr(token html.Token, attr string) (string, error) {
	for _, a := range token.Attr {
		if a.Key == attr { // found it
			return a.Val, nil
		}
	}
	return "", errors.New("Invalid or nonexistent Attribute") // not found
}

// resolveURL takes in a base URL string and relative URL string and returns an absolute URL string
// that can locate the relative resource
func resolveURL(base string, loc string) string {
	bURL, _ := url.Parse(base)
	lURL, _ := url.Parse(loc)

	return bURL.ResolveReference(lURL).String()
}

// buildIcon takes in a token and pageURL to construct and return a
// PreviewImage struct using the attributes in the token and resolves
// any relative URLs using the pageURL
func buildIcon(token html.Token, pageURL string) *PreviewImage {
	rel, _ := getAttr(token, "rel")
	temp := &PreviewImage{}
	if rel == "icon" {
		href, _ := getAttr(token, "href")
		typ, _ := getAttr(token, "type")
		alt, _ := getAttr(token, "alt")
		sizes, _ := getAttr(token, "sizes")

		if !strings.HasPrefix(href, "http") {
			href = resolveURL(pageURL, href)
		}
		temp.URL = href
		temp.Alt = alt
		temp.Type = typ

		if sizes != "any" && sizes != "" {
			sizeSlice := strings.Split(sizes, "x")
			temp.Height, _ = strconv.Atoi(sizeSlice[0])
			temp.Width, _ = strconv.Atoi(sizeSlice[1])
		}
	}
	return temp
}

// buildImage takes in a property string, pageURL, PreviewImage struct and content string to add to the
// PreviewImage using the content in the cont string based on what property it belongs to and resolves
// any relative URLs using the pageURL
func buildImage(prop string, pageURL string, image *PreviewImage, cont string) *PreviewImage {
	if prop == "og:image" {
		if !strings.HasPrefix(cont, "http") { // not absolute path
			cont = resolveURL(pageURL, cont)
		}
		image.URL = cont
	}

	if prop == "og:image:secure_url" {
		image.SecureURL = resolveURL(pageURL, cont)
	} else if prop == "og:image:type" {
		image.Type = cont
	} else if prop == "og:image:width" {
		image.Width, _ = strconv.Atoi(cont)
	} else if prop == "og:image:height" {
		image.Height, _ = strconv.Atoi(cont)
	} else if prop == "og:image:alt" {
		image.Alt = cont
	}
	return image
}
