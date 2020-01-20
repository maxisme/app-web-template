package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Page struct {
	Name    string
	Content string
}

type Sparkle struct {
	Description string
	Version     string `validate:"nonzero"`
}

type ProjectData struct {
	Project     string  `validate:"nonzero"`
	KeyWords    string  `validate:"nonzero"`
	Description string  `validate:"nonzero"`
	Recaptcha   string  `validate:"nonzero"`
	Host        string  `validate:"nonzero"`
	DmgPath     string  `validate:"nonzero"`
	Sparkle     Sparkle `validate:"nonzero"`

	Pages []Page
	Year  int
}

func filenameWithoutExtension(fn string) string {
	return strings.TrimSuffix(path.Base(fn), path.Ext(fn))
}

func renderDirectory(pattern string) []Page {
	var pages []Page

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
		pages = append(pages, Page{
			Name:    filenameWithoutExtension(file),
			Content: tpl.String(),
		})
	}
	return pages
}

func (p *ProjectData) WebHandler(w http.ResponseWriter, r *http.Request) {
	tmplPath := "index.html"
	tmpl := template.Must(template.ParseFiles(tmplPath))

	// fetch all templates in template directory
	p.Pages = renderDirectory("templates/*.html")
	p.Year = time.Now().Year()

	if err := tmpl.Execute(w, p); err != nil {
		log.Fatal(err.Error())
	}
}

func (p *ProjectData) SiteMapHandler(w http.ResponseWriter, r *http.Request) {
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

func (p *ProjectData) VersionHandler(w http.ResponseWriter, r *http.Request) {
	xml := fmt.Sprintf(`<?xml version="1.1" encoding="utf-8"?>
	<rss version="1.1" xmlns:sparkle="https://%[1]s/xml-namespaces/sparkle" xmlns:dc="https://%[1]s/dc/elements/1.1/">
	  <channel>
		<item>
			<title>Version %[2]s</title>
			<description><![CDATA[
				%[3]s
			]]>
			</description>
			<sparkle:version>%[2]s</sparkle:version>
			<pubDate>'.date ("r", filemtime($file)).'</pubDate>
			<enclosure url="https://%[1]s/download" sparkle:version="%[2]s"/>
		</item>
	  </channel>
	</rss>
	`, p.Host, p.Sparkle.Version, p.Sparkle.Description)

	if _, err := w.Write([]byte(xml)); err != nil {
		log.Fatal(err.Error())
	}
}

func (p *ProjectData) DownloadHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Location", p.DmgPath)
}
