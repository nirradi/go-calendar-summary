// Copyright 2011 Google Inc. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	//"time"
	calendar "google.golang.org/api/calendar/v3"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// Flags
var (
	clientID     = flag.String("clientid", "", "OAuth 2.0 Client ID.  If non-empty, overrides --clientid_file")
	clientIDFile = flag.String("clientid-file", "clientid.dat",
		"Name of a file containing just the project's OAuth 2.0 Client ID from https://developers.google.com/console.")
	secret     = flag.String("secret", "", "OAuth 2.0 Client Secret.  If non-empty, overrides --secret_file")
	secretFile = flag.String("secret-file", "clientsecret.dat",
		"Name of a file containing just the project's OAuth 2.0 Client Secret from https://developers.google.com/console.")
	cacheToken = flag.Bool("cachetoken", true, "cache the OAuth 2.0 token")
	debug      = flag.Bool("debug", false, "show HTTP traffic")
)

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: go-api-demo <api-demo-name> [api name args]\n\nPossible APIs:\n\n")
	for n := range demoFunc {
		fmt.Fprintf(os.Stderr, "  * %s\n", n)
	}
	os.Exit(2)
}

func main() {
	flag.Parse()

	randState := "abc123" //fmt.Sprintf("st%d", time.Now().UnixNano())
	log.Printf("Randomstring is %s", randState)
	config := &oauth2.Config{
		ClientID:     valueOrFileContents(*clientID, *clientIDFile),
		ClientSecret: valueOrFileContents(*secret, *secretFile),
		Endpoint:     google.Endpoint,
		Scopes:       []string{calendar.CalendarScope},
	}


	startServer(randState, config)

}

var (
	demoFunc  = make(map[string]func(*http.Client, []string))
	demoScope = make(map[string]string)
)

func registerDemo(name, scope string, main func(c *http.Client, argv []string)) {
	if demoFunc[name] != nil {
		panic(name + " already registered")
	}
	demoFunc[name] = main
	demoScope[name] = scope
}

func osUserCacheDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(os.Getenv("HOME"), "Library", "Caches")
	case "linux", "freebsd":
		return filepath.Join(os.Getenv("HOME"), ".cache")
	}
	log.Printf("TODO: osUserCacheDir on GOOS %q", runtime.GOOS)
	return "."
}


func newOAuthClient(ctx context.Context, config *oauth2.Config, ch <-chan string, randState string) *http.Client {
	config.RedirectURL = "http://127.0.0.1:37555"
	authURL := config.AuthCodeURL(randState)
	go openURL(authURL)
	log.Printf("Authorize this app at: %s", authURL)
	code := <-ch
	log.Printf("Got code: %s", code)

	token, _ := config.Exchange(ctx, code)

	return config.Client(ctx, token)
}

type serverFunc func(rw http.ResponseWriter, req *http.Request)

func httpHandler(randState string, config *oauth2.Config) serverFunc{
	clientTokens := make(map[string]*http.Client)

	ctx := context.Background()

	if *debug {
		ctx = context.WithValue(ctx, oauth2.HTTPClient, &http.Client{
			Transport: &logTransport{http.DefaultTransport},
		})
	}

	return func(rw http.ResponseWriter, req *http.Request) {


		if code := req.FormValue("code"); code != "" {

		/*	if req.FormValue("state") != randState {
				log.Printf("State doesn't match: req = %#v", req)
				http.Error(rw, "", 500)
				return
			}*/

			c, ok := clientTokens[code]
			if !ok {
				token, err := config.Exchange(ctx, code)
				if err != nil {
					log.Printf("there was an error getting a token: %v", err)
				}
				clientTokens[code] = config.Client(ctx, token)
				c = clientTokens[code]
				log.Printf("getting a new client")
			} else {
				log.Printf("get existing client")
			}


			if calendar := req.FormValue("calendar"); calendar != "" {
				fmt.Fprintf(rw, "<html><body><ul>")
				eventBuckets := getEventSummary(c, calendar)
				summary := summarizeEvents(eventBuckets)
				for key, value := range summary {
					fmt.Fprintf(rw, "<li> <label>%s</label> <p>%s</p> </li>", key, value)
				}
				fmt.Fprintf(rw, "</ul></body></html>")

			} else {
				calendars := getCalendars(c)
				fmt.Fprintf(rw, "<html><body><ul>")
				for _, v := range calendars {
					fmt.Fprintf(rw, "<li><a href=\"%s&calendar=%s\">%s</a></li>", req.URL.RequestURI(), v, v)
				}
				fmt.Fprintf(rw, "</ul></body></html>")
			}

			rw.(http.Flusher).Flush()
			return
		}

		config.RedirectURL = "http://127.0.0.1:37555"
		authURL := config.AuthCodeURL(randState)

		fmt.Fprintf(rw, "<a href=\"%s\">Give me permission</a>", authURL)
		rw.(http.Flusher).Flush()
		return
	}
}



func startServer(randState string, config *oauth2.Config) {

	CUSTOM_URL := "0.0.0.0:37555"

	http.HandleFunc("/", httpHandler(randState, config))
	http.ListenAndServe(CUSTOM_URL, nil)

	return
}


func openURL(url string) {
	try := []string{"xdg-open", "google-chrome", "open"}
	for _, bin := range try {
		err := exec.Command(bin, url).Run()
		if err == nil {
			return
		}
	}
	log.Printf("Error opening URL in browser.")
}

func valueOrFileContents(value string, filename string) string {
	if value != "" {
		return value
	}
	slurp, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatalf("Error reading %q: %v", filename, err)
	}
	return strings.TrimSpace(string(slurp))
}