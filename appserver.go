package appserver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/go-systemd/activation"
	"github.com/gorilla/mux"
	"github.com/tylerb/graceful"
	"gopkg.in/gomail.v2"
	"gopkg.in/validator.v2"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
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
	To            string `validate:"nonzero"`
	gomail.Dialer `validate:"nonzero"`
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
	Success            bool     `json:"success"`
}

func (p *ProjectConfig) validateReCAPTCHA(recaptchaResponse string) (bool, error) {
	// https://developers.google.com/recaptcha/docs/verify
	req, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   {p.Recaptcha.Priv},
		"response": {recaptchaResponse},
	})
	if err != nil { // Handle error from HTTP POST to Google reCAPTCHA verify server
		return false, err
	}
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body) // Read the response from Google
	if err != nil {
		return false, err
	}

	var googleResponse GoogleRecaptchaResponse
	err = json.Unmarshal(body, &googleResponse) // Parse the JSON response from Google
	if err != nil {
		return false, err
	}
	return googleResponse.Success, nil
}

func (p *ProjectConfig) emailHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Invalid Request", 404)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), 490)
		return
	}

	senderEmailAddress :=r.FormValue("from")
	senderName := r.FormValue("name")
	body := r.FormValue("body")

	m := gomail.NewMessage()
	m.SetHeader("To", "max@max.me.uk")
	m.SetHeader("From", senderEmailAddress)
	m.SetHeader("Subject", fmt.Sprintf("%s contact form - from %s", p.Name, senderName))
	m.SetBody("text/html", body)

	d := gomail.NewDialer("smtp.example.com", 587, "user", "123456")

	// Send the email to Bob, Cora and Dan.
	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}
}

//<?php
//session_start();
///******************/
///* init VARIABLES */
///******************/
//$settings = yaml_parse_file("../../config.yaml");
//$APP_NAME = $settings["title"];
//$to_email = $settings["email"];
//$CAPTCHA_SECRET = $settings["recaptcha"]["priv"]; /* GET KEYS: https://www.google.com/recaptcha/admin#list */
//
//function diee($mess){
//$_SESSION["error"] = $mess;
//$_SESSION["name"] = $_POST['name'];
//$_SESSION["from"] = $_POST['from'];
//$_SESSION["body"] = $_POST['body'];
//die(header("Location: ../#contact"));
//}
///******************/
///* handle RECAPTCHA */
///******************/
//$userIP = $_SERVER["REMOTE_ADDR"];
//$recaptchaResponse = $_GET['g-recaptcha-response'];
//$request = file_get_contents("https://www.google.com/recaptcha/api/siteverify?secret={$CAPTCHA_SECRET}&response={$recaptchaResponse}&remoteip={$userIP}");
//if(!strstr($request, "true")) diee("Failed captcha");
//
///******************/
///* handle EMAIL */
///******************/
///* REMEMBER TO CHECK VPS ALLOWS 587 OUTGOING */
//
///* POST VALIDATION*/
//if(!filter_var($_POST['from'], FILTER_VALIDATE_EMAIL)) diee("Not a valid email");
//$from = $_POST['from'];
//
//if(!isset($_POST['name'])) diee("What is your name?");
//if(strlen($_POST['name']) > 200) diee("Your name is too long!");
//$name = filter_var($_POST['name'], FILTER_SANITIZE_STRING, FILTER_FLAG_STRIP_HIGH);
//
//if(!isset($_POST['body'])) diee("You haven't told us anything!");
//if(strlen($_POST['body']) > 10000) diee("Your message is too long! Press back if you have lost your essay!");
//$body = filter_var($_POST['body'], FILTER_SANITIZE_STRING, FILTER_FLAG_STRIP_HIGH);
//
//use PHPMailer\PHPMailer\PHPMailer;
//
//require 'PHPMailer/Exception.php';
//require 'PHPMailer/PHPMailer.php';
//require 'PHPMailer/SMTP.php';
//
//$mail = new PHPMailer();
////$mail->SMTPDebug = 4;
//$mail->CharSet = 'UTF-8';
//
//$mail->IsSMTP();
//$mail->Timeout    = 2;
//$mail->Host       = "mail.maxemails.info";
//$mail->SMTPAuth   = true;
//$mail->Username   = "form@transferme.it";
//$mail->Password   = "***REMOVED***";
//$mail->SMTPSecure = 'TLS';
//$mail->Port       = 587;
//
//$mail->setFrom($from, "$from");
//$mail->addAddress($to_email, '');
//$mail->Subject  = "$APP_NAME Contact form - from $name";
//$mail->Body     = $body;
//
//$mail->SMTPOptions = array( 'ssl' => array( 'verify_peer' => false, 'verify_peer_name' => false, 'allow_self_signed' => true ) );
//
//if(!$mail->send()) {
//echo 'Message was not sent.';
//echo 'Mailer error: ' . $mail->ErrorInfo;
//diee("Please contact <a href='mailto:max@max.me.uk'>max@max.me.uk</a>");
//} else {
//$_SESSION["email"] = "Thank you for your message. We will get back to you ASAP.";
//header("Location: ../#contact");
//}
//
//$mail->SmtpClose();

type GoogleRecaptchaResponse struct {
	Success bool `json:"success"`
}

func (p *ProjectConfig) validateReCAPTCHA(recaptchaResponse string) (bool, error) {
	// https://developers.google.com/recaptcha/docs/verify
	req, err := http.PostForm("https://www.google.com/recaptcha/api/siteverify", url.Values{
		"secret":   {p.Recaptcha.Priv},
		"response": {recaptchaResponse},
	})
	if err != nil { // Handle error from HTTP POST to Google reCAPTCHA verify server
		return false, err
	}
	defer req.Body.Close()
	body, err := ioutil.ReadAll(req.Body) // Read the response from Google
	if err != nil {
		return false, err
	}

	var googleResponse GoogleRecaptchaResponse
	err = json.Unmarshal(body, &googleResponse) // Parse the JSON response from Google
	if err != nil {
		return false, err
	}
	return googleResponse.Success, nil
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

	senderEmailAddress := r.FormValue("from")
	if len(senderEmailAddress) > 254 || !rxEmail.MatchString(senderEmailAddress) {
		http.Error(w, "invalid email address", 491)
		return
	}

	senderName := r.FormValue("name")
	body := r.FormValue("body")

	if len(body) <= 5 || len(senderName) < 2 {
		http.Error(w, "invalid data", 492)
		return
	}

	m := gomail.NewMessage()
	m.SetHeader("To", p.Email.To)
	m.SetHeader("From", senderEmailAddress)
	m.SetHeader("Subject", fmt.Sprintf("%s contact form - from %s", p.Name, senderName))
	m.SetBody("text/html", body)

	if err := p.Email.DialAndSend(m); err != nil {
		panic(err)
	}
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
	m := mux.NewRouter()
	m.HandleFunc("/", p.webHandler)
	m.HandleFunc("/sitemap", p.siteMapHandler)
	m.HandleFunc("/version", p.versionHandler)
	m.HandleFunc("/download", p.downloadHandler)
	m.HandleFunc("/email", p.downloadHandler)
	m.PathPrefix("/images/").Handler(http.FileServer(http.Dir(".")))
	m.PathPrefix("/").Handler(http.FileServer(http.Dir(basepath + "/static/")))

	listeners, err := activation.Listeners()
	if err == nil && len(listeners) == 1 {
		return graceful.Serve(&http.Server{Handler: m}, listeners[0], 5*time.Second)
	}

	return graceful.ListenAndServe(&http.Server{Addr: ":9000", Handler: m}, 5*time.Second)
}
