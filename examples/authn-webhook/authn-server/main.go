package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"

	"github.com/yubo/apiserver/pkg/apis/authentication"
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

		token := &authentication.TokenReview{}
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(token); err != nil {
			fmt.Printf("decode err %s", err)
			return
		}

		token.Status.User = authentication.UserInfo{
			Username: "test",
			UID:      "u110",
			Groups:   []string{"test:webhook"},
		}
		token.Status.Audiences = []string{"api"}
		token.Status.Authenticated = true

		w.Header().Set("Content-Type", "application/json")
		encoder := json.NewEncoder(w)
		encoder.Encode(token)
	})

	fmt.Printf("Listening %s ...\n", *port)
	http.ListenAndServe(":"+*port, nil)
}
