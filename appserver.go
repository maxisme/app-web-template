package appserver

import (
	"bytes"
	"fmt"
	"github.com/TV4/graceful"
	"github.com/gorilla/mux"
	"gopkg.in/validator.v2"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type page struct {
	Name    string
	Content string
}

type Sparkle struct {
	Description string
	Version     float32 `validate:"nonzero"`
}

type Recaptcha struct {
	Pub  string `validate:"nonzero"`
	Priv string `validate:"nonzero"`
}

type ProjectData struct {
	Project     string    `validate:"nonzero"`
	KeyWords    string    `validate:"nonzero"`
	Description string    `validate:"nonzero"`
	Recaptcha   Recaptcha `validate:"nonzero"`
	Host        string    `validate:"nonzero"`
	DmgPath     string    `validate:"nonzero"`
	Sparkle     Sparkle   `validate:"nonzero"`

	pages []page
	year  int
}

func filenameWithoutExtension(fn string) string {
	return strings.TrimSuffix(path.Base(fn), path.Ext(fn))
}

func renderDirectory(pattern string) []page {
	var pages []page

	files, err := filepath.Glob(pattern)
	if err != nil {
		log.Fatal(err.Error())
	}

	for _, file := range files {
		ts, err := template.ParseFiles(file)
		if err != nil {
			log.Fatal(err.Error())
		}
		var tpl bytes.Buffer
		if err := ts.Execute(&tpl, nil); err != nil {
			log.Fatal(err.Error())
		}
		pages = append(pages, page{
			Name:    filenameWithoutExtension(file),
			Content: tpl.String(),
		})
	}
	return pages
}

func (p *ProjectData) webHandler(w http.ResponseWriter, r *http.Request) {
	tmplPath := "index.html"
	tmpl := template.Must(template.ParseFiles(tmplPath))

	// fetch all templates in template directory
	p.pages = renderDirectory("templates/*.html")
	p.year = time.Now().Year()

	if err := tmpl.Execute(w, p); err != nil {
		log.Fatal(err.Error())
	}
}

func (p *ProjectData) siteMapHandler(w http.ResponseWriter, r *http.Request) {
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
	<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
	  <loc>https://%s/</loc>
	  <lastmod>2019-12-21T00:32:56+00:00</lastmod>
	  <changefreq>monthly</changefreq>
	</url>
	</urlset>
	`, p.Host)

	if _, err := w.Write([]byte(xml)); err != nil {
		log.Fatal(err.Error())
	}
}

func (p *ProjectData) versionHandler(w http.ResponseWriter, r *http.Request) {
	xml := fmt.Sprintf(`<?xml version="1.1" encoding="utf-8"?>
	<rss version="1.1" xmlns:Sparkle="https://%[1]s/xml-namespaces/Sparkle" xmlns:dc="https://%[1]s/dc/elements/1.1/">
	  <channel>
		<item>
			<title>Version %[2]s</title>
			<description><![CDATA[
				%[3]s
			]]>
			</description>
			<Sparkle:version>%[2]s</Sparkle:version>
			<pubDate>'.date ("r", filemtime($file)).'</pubDate>
			<enclosure url="https://%[1]s/download" Sparkle:version="%[2]s"/>
		</item>
	  </channel>
	</rss>
	`, p.Host, p.Sparkle.Version, p.Sparkle.Description)

	if _, err := w.Write([]byte(xml)); err != nil {
		log.Fatal(err.Error())
	}
}

func (p *ProjectData) downloadHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Location", p.DmgPath)
}

func Start(p ProjectData) {
	// validate requirements to start server
	if err := validator.Validate(p); err != nil {
		_ = fmt.Errorf(err.Error())
		return
	}
	if _, err := os.Stat("/images"); os.IsNotExist(err) {
		_ = fmt.Errorf("no /images directory")
		return
	}
	if _, err := os.Stat("/templates"); os.IsNotExist(err) {
		_ = fmt.Errorf("no /templates directory")
		return
	}
	if _, err := os.Stat(p.DmgPath); os.IsNotExist(err) {
		_ = fmt.Errorf("no dmg at path")
		return
	}

	m := mux.NewRouter()
	m.HandleFunc("/", p.webHandler)
	m.HandleFunc("/sitemap", p.siteMapHandler)
	m.HandleFunc("/version", p.versionHandler)
	m.HandleFunc("/download", p.downloadHandler)
	m.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	m.PathPrefix("/").Handler(http.FileServer(http.Dir("./images/")))
	graceful.ListenAndServe(&http.Server{Addr: ":8080", Handler: m})
}
