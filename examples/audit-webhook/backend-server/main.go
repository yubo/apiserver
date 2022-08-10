package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
)

func main() {
	port := flag.String("p", "8081", "listen port")

	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, _ := httputil.DumpRequest(r, true)
		fmt.Printf("%s\n", string(b))
	})

	fmt.Printf("Listening %s ...\n", *port)
	http.ListenAndServe(":"+*port, nil)
}
