package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/yubo/apiserver/pkg/apis/authorization"
)

type response struct {
	Status struct {
		Allowed bool `json:"allowed"`
		Denied  bool `json:"denied"`
	} `json:"status"`
}

func main() {
	port := flag.String("p", "8081", "listen port")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		b, _ := httputil.DumpRequest(r, true)
		fmt.Printf("%s\n", string(b))

		w.Header().Set("Content-Type", "application/json")

		encoder := json.NewEncoder(w)
		encoder.Encode(authorization.SubjectAccessReview{
			Status: authorization.SubjectAccessReviewStatus{
				Allowed: true,
			},
		})
	})

	fmt.Printf("Listening %s ...\n", *port)
	http.ListenAndServe(":"+*port, nil)
}
