package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type Page struct {
	Name    string
	Content string
}

type WebHandlerData struct {
	Project     string
	KeyWords    string
	Description string
	Recaptcha   string

	Pages []Page

	Year int
}

func Getenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("environment variable not set: " + key)
		return ""
	}
	return v
}

func FilenameWithoutExtension(fn string) string {
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
			Name:    FilenameWithoutExtension(file),
			Content: tpl.String(),
		})
	}
	return pages
}

func WebHandler(w http.ResponseWriter, r *http.Request) {
	tmplPath := "index.html"
	tmpl := template.Must(template.ParseFiles(tmplPath))

	// fetch all templates in template directory
	pages := renderDirectory("templates/*.html")

	if err := tmpl.Execute(w, WebHandlerData{
		Project:     Getenv("project"),
		KeyWords:    Getenv("meta-keywords"),
		Description: Getenv("description"),
		Recaptcha:   Getenv("recaptcha-pub"),

		Pages: pages,
		Year:  time.Now().Year(),
	}); err != nil {
		log.Fatal(err.Error())
	}
}

func SiteMapHandler(w http.ResponseWriter, r *http.Request) {
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
	<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
	  <loc>https://%s/</loc>
	  <lastmod>2019-12-21T00:32:56+00:00</lastmod>
	  <changefreq>monthly</changefreq>
	</url>
	</urlset>
	`, Getenv("host"))

	if _, err := w.Write([]byte(xml)); err != nil {
		log.Fatal(err.Error())
	}
}

func VersionHandler(w http.ResponseWriter, r *http.Request) {
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
	`, Getenv("host"), Getenv("version"), Getenv("sparkle-description"))

	if _, err := w.Write([]byte(xml)); err != nil {
		log.Fatal(err.Error())
	}
}

func DownloadHandler(w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Location", 	Getenv("dmg-path"))
}
