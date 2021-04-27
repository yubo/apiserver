package urlencoded

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"io/ioutil"

	restful "github.com/emicklei/go-restful"
	"github.com/stretchr/testify/require"
)

func TestUrlEncoded(t *testing.T) {

	// register url encoded entity
	restful.RegisterEntityAccessor(MIME_URL_ENCODED, NewEntityAccessor())
	type Tool struct {
		Name   string
		Vendor string
	}

	// Write
	httpWriter := httptest.NewRecorder()
	in := &Tool{Name: "json", Vendor: "apple"}
	out := &Tool{}
	resp := restful.NewResponse(httpWriter)
	resp.SetRequestAccepts(MIME_URL_ENCODED)

	err := resp.WriteEntity(in)
	if err != nil {
		t.Errorf("err %v", err)
	}

	// Read
	bodyReader := bytes.NewReader(httpWriter.Body.Bytes())
	httpRequest, _ := http.NewRequest("POST", "/test", bodyReader)
	httpRequest.Header.Set("Content-Type", MIME_URL_ENCODED)
	request := restful.NewRequest(httpRequest)
	err = request.ReadEntity(out)
	if err != nil {
		t.Errorf("err %v", err)
	}

	require.Equal(t, in, out)
}

func TestWithWebService(t *testing.T) {
	serverURL := "http://127.0.0.1:8090"
	go func() {
		runRestfulUrlEncodedRouterServer()
	}()
	if err := waitForServerUp(serverURL); err != nil {
		t.Errorf("%v", err)
	}

	// send a post request
	userData := user{Id: "0001", Name: "Tony"}
	urlEncodedData, err := Marshal(userData)
	req, err := http.NewRequest("POST", serverURL+"/test/urlencoded", bytes.NewBuffer(urlEncodedData))
	req.Header.Set("Content-Type", MIME_URL_ENCODED)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("unexpected error in sending req: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("unexpected response: %v, expected: %v", resp.StatusCode, http.StatusOK)
	}

	ur := &userResponse{}
	expectUrlEncodedDocument(t, resp, ur)
	if ur.Status != statusActive {
		t.Fatalf("should not error")
	}
	log.Printf("user response:%v", ur)
}

func expectUrlEncodedDocument(t *testing.T, r *http.Response, doc interface{}) {
	data, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		t.Errorf("ExpectUrlEncodedDocument: unable to read response body :%v", err)
		return
	}
	// put the body back for re-reads
	r.Body = ioutil.NopCloser(bytes.NewReader(data))

	err = Unmarshal(data, doc)
	if err != nil {
		t.Errorf("ExpectUrlEncodedDocument: unable to unmarshal UrlEncoded:%v", err)
	}
}

func runRestfulUrlEncodedRouterServer() {

	container := restful.NewContainer()
	register(container)

	log.Print("start listening on localhost:8090")
	server := &http.Server{Addr: ":8090", Handler: container}
	log.Fatal(server.ListenAndServe())
}

func waitForServerUp(serverURL string) error {
	for start := time.Now(); time.Since(start) < time.Minute; time.Sleep(5 * time.Second) {
		_, err := http.Get(serverURL + "/")
		if err == nil {
			return nil
		}
	}
	return errors.New("waiting for server timed out")
}

var (
	statusActive = "active"
)

type user struct {
	Id, Name string
}

type userResponse struct {
	Status string
}

func register(container *restful.Container) {
	restful.RegisterEntityAccessor(MIME_URL_ENCODED, NewEntityAccessor())
	ws := new(restful.WebService)
	ws.
		Path("/test").
		Consumes(restful.MIME_JSON, MIME_URL_ENCODED).
		Produces(restful.MIME_JSON, MIME_URL_ENCODED)
	// route user api
	ws.Route(ws.POST("/urlencoded").
		To(do).
		Reads(user{}).
		Writes(userResponse{}))
	container.Add(ws)
}

func do(request *restful.Request, response *restful.Response) {
	u := &user{}
	err := request.ReadEntity(u)
	if err != nil {
		log.Printf("should be no error, got:%v", err)
	}
	log.Printf("got:%v", u)

	ur := &userResponse{Status: statusActive}

	response.SetRequestAccepts(MIME_URL_ENCODED)
	response.WriteEntity(ur)
}
