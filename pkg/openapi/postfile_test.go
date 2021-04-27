package openapi

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yubo/golib/util"
)

func genFile(data []byte) (string, error) {
	return util.WriteTempFile("/tmp", "*.test.txt", data)
}

func testPostFile(in, out interface{}) error {
	b, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func testEqualFile(t *testing.T, name, file string, content []byte) {
	defer os.Remove(file)

	b, err := ioutil.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s file %s content %s", name, file, string(content))
	require.Equal(t, content, b, name)
}

func TestPostFile(t *testing.T) {
	content := []byte("hello world")

	file, err := genFile(content)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file)

	// postfile
	{
		in, out := PostFile{FileName: util.String(file)}, PostFile{}
		err = testPostFile(&in, &out)
		if err != nil {
			t.Fatal(err)
		}
		testEqualFile(t, "*postfile", out.String(), content)

		err = testPostFile(in, &out)
		if err != nil {
			t.Fatal(err)
		}
		testEqualFile(t, "postfile", out.String(), content)
	}

	{
		in := PostFiles{
			PostFile{FileName: util.String(file)},
			PostFile{FileName: util.String(file)},
		}
		out := PostFiles{}
		err = testPostFile(&in, &out)
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range out {
			testEqualFile(t, fmt.Sprintf("postfiles %d", k), v.String(), content)
		}
	}

}
