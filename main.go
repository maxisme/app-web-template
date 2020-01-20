package main

import (
	"fmt"
	"github.com/TV4/graceful"
	"github.com/gorilla/mux"
	"gopkg.in/validator.v2"
	"net/http"
	"os"
)

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

	mux := mux.NewRouter()
	mux.HandleFunc("/", p.WebHandler)
	mux.HandleFunc("/sitemap", p.SiteMapHandler)
	mux.HandleFunc("/version", p.VersionHandler)
	mux.HandleFunc("/download", p.DownloadHandler)
	mux.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	mux.PathPrefix("/").Handler(http.FileServer(http.Dir("./images/")))
	graceful.ListenAndServe(&http.Server{Addr: ":8080", Handler: mux})
}
