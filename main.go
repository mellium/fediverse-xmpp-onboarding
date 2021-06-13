// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

// The fediverse-xmpp-onboarding authorizes with the fediverse and provisions
// XMPP accounts.
//
// It is mostly meant as an example of the pre-auth key generation protoXEP and
// is likely not usable in the real world.
package main

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"code.soquee.net/mux"
	"github.com/mattn/go-mastodon"
)

const (
	verifiedPage = "/verified"
)

func parseTemplates() (*template.Template, error) {
	return template.New("root").Funcs(tmplFuncs()).ParseFS(tmpls, pagesGlob)
}

func main() {
	// Setup logging
	logger := log.New(os.Stderr, "", log.LstdFlags)
	debug := log.New(io.Discard, "DEBUG ", log.LstdFlags)

	// Setup and parse command line flags
	var (
		verbose     = false
		mastoServer = "http://127.0.0.1:4000"
		listen      = "127.0.0.1:8080"
		base        = "http://" + listen

		ibr      bool
		secret   string
		hostname string
	)
	flags := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	flags.BoolVar(&verbose, "v", verbose, "enable verbose logging")
	flags.BoolVar(&ibr, "ibr", ibr, "set if the server supports in-band registration")
	flags.StringVar(&mastoServer, "mastodon", mastoServer, "Mastodon or Pleroma server")
	flags.StringVar(&base, "base", base, "URL of this service")
	flags.StringVar(&listen, "listen", listen, "interface on which to listen for HTTP requests")
	flags.StringVar(&secret, "secret", "", "the shared secret used to generate XMPP invites (required)")
	flags.StringVar(&hostname, "host", "", "the XMPP server to generate invites for (required)")
	err := flags.Parse(os.Args[1:])
	if err != nil {
		logger.Fatalf("error parsing command-line flags: %v", err)
	}

	switch "" {
	case hostname:
		logger.Fatalf("the -host flag is required, see %s -help for usage", os.Args[0])
	case secret:
		logger.Fatalf("the -secret flag is required, see %s -help for usage", os.Args[0])
	}

	// Configure logging
	if verbose {
		debug.SetOutput(os.Stderr)
	}

	// Load page templates
	parsedTemplates, err := parseTemplates()
	if err != nil {
		logger.Fatalf("error parsing internal templates: %v", err)
	}

	app, err := mastodon.RegisterApp(context.Background(), &mastodon.AppConfig{
		Server:       mastoServer,
		ClientName:   os.Args[0],
		Scopes:       "read",
		Website:      base,
		RedirectURIs: base + verifiedPage,
	})
	if err != nil {
		logger.Fatalf("error connecting to Mastodon or Pleroma at %s: %v", mastoServer, err)
	}

	baseURL, err := url.Parse(base)
	if err != nil {
		logger.Fatalf("base address %q is not a valid URL: %v", base, err)
	}

	serveMux := mux.New(
		mux.Handle(http.MethodGet, "/", renderTmpl(app, baseURL, parsedTemplates, logger, debug)),
		mux.Handle(http.MethodGet, verifiedPage, createAccount(ibr, secret, hostname, mastoServer, app, baseURL, parsedTemplates, logger, debug)),
		mux.Handle(http.MethodGet, "/static/{p path}", http.StripPrefix("/static/", http.FileServer(http.FS(assets)))),
	)

	server := &http.Server{
		Addr:           listen,
		Handler:        http.StripPrefix(baseURL.Path, serveMux),
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	logger.Printf("listening on %s…", listen)
	logger.Fatal(server.ListenAndServe())
}

// Page is the data passed to the template rendering for every page.
type Page struct {
	InstanceName string
	BaseURL      *url.URL
	App          *mastodon.Application

	Data interface{}
}

type renderFunc func(http.ResponseWriter, *http.Request, interface{}) error

func renderer(app *mastodon.Application, baseURL *url.URL, tmpl *template.Template, logger, debug *log.Logger, data func(Page) interface{}) renderFunc {
	if data == nil {
		data = func(p Page) interface{} {
			return p
		}
	}
	return func(w http.ResponseWriter, r *http.Request, extraData interface{}) error {
		var err error
		tmpl, err = reparseTemplates(tmpl)
		if err != nil {
			logger.Printf("error reparsing templates: %v", err)
		}

		name := r.URL.Path
		name = strings.TrimPrefix(name, "/")
		if name == "" {
			name = "index"
		}
		name += ".html"
		debug.Printf("rendering template %s…", name)
		return tmpl.ExecuteTemplate(w, name, data(Page{
			InstanceName: "test",
			BaseURL:      baseURL,
			App:          app,

			Data: extraData,
		}))
	}
}

func renderTmpl(app *mastodon.Application, baseURL *url.URL, tmpl *template.Template, logger, debug *log.Logger) http.HandlerFunc {
	r := renderer(app, baseURL, tmpl, logger, debug, nil)

	return func(w http.ResponseWriter, req *http.Request) {
		err := r(w, req, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func createAccount(ibr bool, secret, hostname, mastoServer string, app *mastodon.Application, baseURL *url.URL, tmpl *template.Template, logger, debug *log.Logger) http.HandlerFunc {
	r := renderer(app, baseURL, tmpl, logger, debug, nil)

	return func(w http.ResponseWriter, req *http.Request) {
		tok := req.FormValue("code")
		c := mastodon.NewClient(&mastodon.Config{
			Server:       mastoServer,
			ClientID:     app.ClientID,
			ClientSecret: app.ClientSecret,
		})
		ctx := req.Context()
		err := c.AuthenticateToken(ctx, tok, app.RedirectURI)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		userAccount, err := c.GetAccountCurrentUser(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		useIBR := "n"
		if ibr {
			useIBR = "y"
		}
		preauthTok := key(secret, hostname, time.Now())
		u, err := url.Parse(fmt.Sprintf(`xmpp:%s?roster;preauth=%s;ibr=%s`, hostname, preauthTok, useIBR))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = r(w, req, struct {
			Key  string
			User *mastodon.Account
			URL  template.URL
		}{
			Key:  preauthTok,
			User: userAccount,
			URL:  template.URL(u.String()),
		})
		if err != nil {
			logger.Printf("error rendering template: %v", err)
		}
	}
}

func key(secret, hostname string, now time.Time) string {
	milliTime := (now.UnixNano() + 1e6 - 1) / 1e6

	h := hmac.New(sha256.New, []byte(secret))
	fmt.Fprintf(h, "%s:%d", hostname, milliTime)

	tok := string(h.Sum(nil))
	tok = base64.RawURLEncoding.EncodeToString([]byte(tok))

	jids := base64.RawURLEncoding.EncodeToString([]byte(hostname))

	return fmt.Sprintf("%s:%s:%d", tok, jids, milliTime)
}
