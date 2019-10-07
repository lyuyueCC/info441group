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

	/**
	Add an HTTP header to the response with the name
	`Access-Control-Allow-Origin` and a value of `*`.
	This will allow cross-origin AJAX requests to your server
	*/
	w.Header().Set("Access-Control-Allow-Origin", "*")

	//Get the `url` query string parameter value from the request.
	//If not supplied, respond with an http.StatusBadRequest error.
	pageURL := r.URL.Query().Get("url")
	if len(pageURL) == 0 {
		http.Error(w, "No query found in the requested url", http.StatusBadRequest)
		return
	}

	//Call fetchHTML() to fetch the requested URL.
	resp, err := fetchHTML(pageURL)
	if err != nil {
		//.Fatalf() prints the error and exists the process
		log.Fatalf("error fetching URL: %v\n", err)
	}

	//Call extractSummary() to extract the page summary meta-data
	PageSummary, err := extractSummary(pageURL, resp)
	if err != nil {
		http.Error(w, fmt.Sprintf("error extracting summary: %v", err), http.StatusBadRequest)
		return
	}

	//Close the response HTML stream so that no resources are leaked.
	defer resp.Close()

	//create a new JSON encoder over stdout and encode the struct into JSON
	w.Header().Add("Content-Type", "application/json")
	json.NewEncoder(w).Encode(PageSummary)
}

//fetchHTML fetches `pageURL` and returns the body stream or an error.
//Errors are returned if the response status code is an error (>=400),
//or if the content type indicates the URL is not an HTML page.
func fetchHTML(pageURL string) (io.ReadCloser, error) {
	//GET the URL
	resp, err := http.Get(pageURL)

	//if there was an error, report it and exit
	if err != nil {
		//.Fatalf() prints the error and exits the process
		log.Fatalf("error fetching URL: %v\n", err)
	}

	//check response status code.
	//if the response status code is >= 400, then return a nil stream and an error.
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("response status code was %d", resp.StatusCode)
	}

	//check response content type.
	//if the response content type does not indicate that the content is a web page, return a nil stream and an error.
	ctype := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ctype, "text/html") {
		return nil, fmt.Errorf("response content type was %s, not text/html", ctype)
	}

	//otherwise return the response body and no (nil) error.
	return resp.Body, err
}

//extractSummary tokenizes the `htmlStream` and populates a PageSummary
//struct with the page's summary meta-data.
func extractSummary(pageURL string, htmlStream io.ReadCloser) (*PageSummary, error) {
	//create an empty instance of a PageSummary to receive the decoded JSON
	summary := &PageSummary{}

	//create a new tokenizer over the response body
	tokenizer := html.NewTokenizer(htmlStream)

	//loop until we find the element
	for {
		//get the next token type
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			err := tokenizer.Err()
			if err == io.EOF {
				//end of the file, break out of the loop
				break
			}
			//otherwise, there was an error tokenizing
			//report the error and exit the process with a non-zero status code.
			log.Fatalf("error tokenizing HTML: %v", tokenizer.Err())
		}

		//get the token
		token := tokenizer.Token()
		//if it's a start tag
		if tokenType == html.StartTagToken || tokenType == html.SelfClosingTagToken {
			//looking for meta tags
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

			//if no Open Graph title
			if token.Data == "title" && summary.Title == "" {
				temp := tokenizer.Next()
				if temp == html.TextToken {
					summary.Title = tokenizer.Token().Data
				}
			}

			//link tags
			if token.Data == "link" {
				summary.Icon = buildIcon(token, pageURL)
			}
		}

		//end after we have parsed head tag
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
	//convert both strings into URLs
	bURL, _ := url.Parse(base)
	lURL, _ := url.Parse(loc)

	//make an absolute path to resource, will ignore if already absolute
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
