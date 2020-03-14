package appserver

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	. "net"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/coreos/go-systemd/activation"
	"github.com/go-chi/chi"
	"github.com/tylerb/graceful"
	"gopkg.in/gomail.v2"
	"gopkg.in/validator.v2"
)

const rfc2822 = "Mon, 28 Jan 2013 14:30:00 +0500"

var requiredPaths = [4]string{
	"images/og_logo.png",
	"images/icon.ico",
	"images/logo.png",
	"templates/",
}

var (
	_, b, _, _ = runtime.Caller(0)
	basepath   = filepath.Dir(b)
)

type page struct {
	Name    string
	Content template.HTML
}

type Sparkle struct {
	Description string `validate:"nonzero"`
	Version     string `validate:"nonzero"`
}

type Email struct {
	To       string `validate:"nonzero"`
	Host     string `validate:"nonzero"`
	Port     int    `validate:"nonzero"`
	Username string `validate:"nonzero"`
	Password string `validate:"nonzero"`
}

type Recaptcha struct {
	Pub  string `validate:"nonzero"`
	Priv string `validate:"nonzero"`
}

type ProjectConfig struct {
	Name        string    `validate:"nonzero"`
	Host        string    `validate:"nonzero"`
	DmgPath     string    `validate:"nonzero"`
	KeyWords    string    `validate:"nonzero"`
	Description string    `validate:"nonzero"`
	Recaptcha   Recaptcha `validate:"nonzero"`
	Sparkle     Sparkle   `validate:"nonzero"`
	Email       Email     `validate:"nonzero"`
}

type IndexData struct {
	Project *ProjectConfig
	Pages   []page
	Year    int
}

type TemplateData struct {
	Data interface{}
}

func filenameWithoutExtension(fn string) string {
	return strings.ToLower(strings.TrimSuffix(path.Base(fn), path.Ext(fn)))
}

func replace(input, from, to string) string {
	return strings.Replace(input, from, to, -1)
}

func renderDirectory(pattern string, data interface{}) []page {
	var pages []page

	files, err := filepath.Glob(pattern)
	if err != nil {
		panic(err.Error())
	}

	for _, file := range files {
		ts, err := template.ParseFiles(file)
		if err != nil {
			panic(err.Error())
		}
		var tpl bytes.Buffer
		if err := ts.Execute(&tpl, TemplateData{data}); err != nil {
			panic(err.Error())
		}
		pages = append(pages, page{
			Name:    filenameWithoutExtension(file),
			Content: template.HTML(tpl.String()),
		})
	}
	return pages
}

type GoogleRecaptchaResponse struct {
	Success bool `json:"success"`
}

func getIP(r *http.Request) string {
	forwarded := r.Header.Get("X-FORWARDED-FOR")
	if forwarded != "" {
		return forwarded
	}
	return r.RemoteAddr
}

func (p *ProjectConfig) isValidCaptcha(recaptchaResponse string, remoteIP string) (bool, error) {
	// https://developers.google.com/recaptcha/docs/verify
	c := &http.Client{Timeout: 1 * time.Second}
	req, err := c.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   {p.Recaptcha.Priv},
		"response": {recaptchaResponse},
		"remoteip": {remoteIP},
	})
	if err != nil {
		return false, err
	}
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return false, err
	}

	var resp GoogleRecaptchaResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		fmt.Printf(string(body))
		return false, err
	}
	return resp.Success, nil
}

var rxEmail = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+\\/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func (p *ProjectConfig) emailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid Request", 404)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 490)
		return
	}

	valid, err := p.isValidCaptcha(r.FormValue("g-recaptcha-response"), getIP(r))
	if err != nil {
		http.Error(w, err.Error(), 491)
		return
	}

	if !valid {
		http.Error(w, "Invalid captcha", 491)
		return
	}

	senderEmailAddress := r.FormValue("from")
	if len(senderEmailAddress) > 254 || !rxEmail.MatchString(senderEmailAddress) {
		http.Error(w, "invalid email address", 492)
		return
	}

	senderName := r.FormValue("name")
	body := r.FormValue("body")

	if len(body) == 0 || len(senderName) == 0 {
		http.Error(w, "invalid form values", 493)
		return
	}

	m := gomail.NewMessage()
	m.SetHeader("To", p.Email.To)
	m.SetHeader("From", senderEmailAddress)
	m.SetHeader("Subject", fmt.Sprintf("%s contact form - from %s", p.Name, senderName))
	m.SetBody("text/html", body)

	d := gomail.NewDialer(p.Email.Host, p.Email.Port, p.Email.Username, p.Email.Password)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}

	http.Redirect(w, r, "/#contact", 301)
}

func (p *ProjectConfig) webHandler(w http.ResponseWriter, r *http.Request) {
	tmplPath := basepath + "/index.html"
	tmpl := template.Must(template.New("index.html").Funcs(template.FuncMap{
		"replace": replace,
	}).ParseFiles(tmplPath))

	var data interface{}

	switch r.Method {
	case "GET":
		data = r.URL.Query()
	case "POST":
		data, _ = ioutil.ReadAll(r.Body)
	default:
		w.WriteHeader(http.StatusNotImplemented)
		_, _ = w.Write([]byte(http.StatusText(http.StatusNotImplemented)))
		return
	}

	if err := tmpl.Execute(w, IndexData{
		Project: p,
		Pages:   renderDirectory("templates/*.html", data),
		Year:    time.Now().Year(),
	}); err != nil {
		panic(err.Error())
	}
}

func (p *ProjectConfig) siteMapHandler(w http.ResponseWriter, r *http.Request) {
	xml := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
	<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
	<url>
	  <loc>https://%s/</loc>
	  <lastmod>2020-01-01T00:00:00+00:00</lastmod>
	  <changefreq>monthly</changefreq>
	</url>
	</urlset>
	`, p.Host)

	if _, err := w.Write([]byte(xml)); err != nil {
		panic(err.Error())
	}
}

func (p *ProjectConfig) versionHandler(w http.ResponseWriter, r *http.Request) {
	// get age of dmg
	info, err := os.Stat(p.DmgPath)
	if err != nil {
		panic(err.Error())
		return
	}
	dmgTime := info.ModTime().Format(rfc2822)

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
			<pubDate>%[4]s</pubDate>
			<enclosure url="https://%[1]s/download" Sparkle:version="%[2]s"/>
		</item>
	  </channel>
	</rss>
	`, p.Host, p.Sparkle.Version, p.Sparkle.Description, dmgTime)

	if _, err := w.Write([]byte(xml)); err != nil {
		panic(err.Error())
	}
}

func (p *ProjectConfig) downloadHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Disposition", "attachment; filename=\""+p.DmgPath+"\"")
	http.ServeFile(w, r, p.DmgPath)
}

// Serve runs the web server based on the config in ProjectConfig
func Serve(p ProjectConfig) error {
	// validate requirements to start server
	if err := validator.Validate(p); err != nil { // validate config
		return err
	}
	for _, requiredPath := range requiredPaths { // insure paths exist
		if _, err := os.Stat(requiredPath); os.IsNotExist(err) {
			return errors.New(requiredPath + " required")
		}
	}
	if _, err := os.Stat(p.DmgPath); os.IsNotExist(err) { // insure valid dmg
		return errors.New(p.DmgPath + "doesn't exist")
	}

	// start server
	m := chi.NewRouter()
	m.HandleFunc("/", p.webHandler)
	m.HandleFunc("/sitemap", p.siteMapHandler)
	m.HandleFunc("/version", p.versionHandler)
	m.HandleFunc("/download", p.downloadHandler)
	m.HandleFunc("/email", p.emailHandler)
	m.Route("/images", func(r chi.Router) {
		r.Handle("/*", http.FileServer(http.Dir(".")))
	})
	m.Handle("/*", http.FileServer(http.Dir(basepath + "/static/")))

	listeners, err := activation.Listeners()
	if err == nil && len(listeners) == 1 {
		return graceful.Serve(&http.Server{Handler: m}, listeners[0], 5*time.Second)
	}

	listener, err := Listen("tcp", "127.0.0.1:0")
	if err != nil {
		panic(err)
	}
	fmt.Println("listening on http://"+listener.Addr().String())
	return graceful.Serve(&http.Server{Handler: m}, listener, 5*time.Second)
}
