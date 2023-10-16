package main

import (
	"flag"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", "7070", "HTTP network address")
	flag.Parse()

	mux := http.NewServeMux()

	fileServer := http.FileServer(http.Dir("./ui/static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fileServer))

	http.HandleFunc("/api/archive/information", ArchiveInformationHandler)
	http.HandleFunc("/api/archive/files", ArchiveFilesHandler)
	mux.HandleFunc("/", HomePage)
	mux.HandleFunc("/upload", UploadFile)

	log.Printf("Server is listening... http://localhost:%s", *addr)
	err := http.ListenAndServe(":"+*addr, mux)
	if err != nil {
		log.Fatal(err)
	}
}
