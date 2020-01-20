package main

import (
	"fmt"
	"net/http"
	"os"
)
import "github.com/TV4/graceful"
import "github.com/gorilla/mux"

func main() {
	// check for directories
	if _, err := os.Stat("/images"); os.IsNotExist(err) {
		_ = fmt.Errorf("no /images directory")
		return
	}
	if _, err := os.Stat("/templates"); os.IsNotExist(err) {
		_ = fmt.Errorf("no /templates directory")
		return
	}

	mux := mux.NewRouter()
	mux.HandleFunc("/", WebHandler)
	mux.HandleFunc("/sitemap", SiteMapHandler)
	mux.HandleFunc("/version", VersionHandler)
	mux.HandleFunc("/download", DownloadHandler)
	mux.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	mux.PathPrefix("/").Handler(http.FileServer(http.Dir("./images/")))
	graceful.ListenAndServe(&http.Server{Addr: ":8080", Handler: mux})
}
