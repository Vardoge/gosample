package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SYNQfm/helpers/test_helper"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	SAMPLE_JOB_JSON = "sample/job.json"
)

func cleanupTestCalls() {
	db.MustExec("TRUNCATE api_calls")
}

func setupTestSynq() {
	url := test_helper.SetupServer()
	sApi.Url = url
	sApi.Key = test_helper.API_KEY
}

func setupTestCalls() (calls []ApiCall) {
	cleanupTestCalls()
	c := ApiCall{VideoId: "123", Type: "details"}
	c.Save()
	calls = append(calls, c)
	c = ApiCall{VideoId: "456", Type: "details"}
	c.Save()
	calls = append(calls, c)
	return calls
}

func setupTest(t *testing.T) *require.Assertions {
	assert := require.New(t)
	cleanupTestCalls()
	setupTestSynq()
	return assert
}

func testLaunchRouter() string {
	port := "32187"
	os.Setenv("PORT", port)
	gin.SetMode(gin.TestMode)
	go setupRouter()
	return port
}

func TestSave(t *testing.T) {
	assert := setupTest(t)
	call := ApiCall{VideoId: "123", Called: time.Now()}
	err := call.Save()
	assert.Nil(err)
	calls := []ApiCall{}
	e := db.Select(&calls, "select * from api_calls where true")
	assert.Nil(e)
	assert.Len(calls, 1)
	assert.Equal(call.VideoId, calls[0].VideoId)
	layout := "2006-01-02 03:04:05"
	assert.Equal(call.Called.UTC().Format(layout), calls[0].Called.UTC().Format(layout))
}

func TestCall(t *testing.T) {

}

func TestGetCalls(t *testing.T) {
	assert := setupTest(t)
	calls, err := getCalls()
	assert.Nil(err)
	assert.Len(calls, 0)
	testCalls := setupTestCalls()
	calls, err = getCalls()
	assert.Nil(err)
	assert.Len(calls, len(testCalls))
}

func TestDetails(t *testing.T) {
	assert := setupTest(t)
	test_helper.RunSimplePost("/v1/details", "",
		func(c *gin.Context) {
			details(c)
		},
		func(r *httptest.ResponseRecorder) {
			assert.Equal(400, r.Code)
			resp := test_helper.ParseJson(r.Body)
			assert.Equal("missing 'video_id'", resp["message"])
		})

	key := sApi.Key
	sApi.Key = "fake"
	test_helper.RunSimplePost("/v1/details", `{"video_id" : "`+test_helper.VIDEO_ID+`"}`,
		func(c *gin.Context) {
			details(c)
		},
		func(r *httptest.ResponseRecorder) {
			assert.Equal(400, r.Code)
			resp := test_helper.ParseJson(r.Body)
			reqs, vals := test_helper.GetReqs()
			assert.Len(reqs, 1)
			assert.Len(vals, 1)
			assert.Equal("failled to make api call : Invalid uuid. Example: '1c0e3ea4529011e6991554a050defa20'.", resp["message"])
		})
	sApi.Key = key
	type Resp struct {
		ServerStarted time.Time `json:"server_started"`
		Calls         []ApiCall `json:"calls"`
		Error         string    `json:"error"`
	}
	test_helper.RunSimplePost("/v1/details", `{"video_id" : "`+test_helper.VIDEO_ID+`"}`,
		func(c *gin.Context) {
			details(c)
		},
		func(r *httptest.ResponseRecorder) {
			var resp Resp
			assert.Equal(200, r.Code)
			bytes, _ := ioutil.ReadAll(r.Body)
			json.Unmarshal(bytes, resp)
			assert.Len(resp.Calls, 0)
			reqs, vals := test_helper.GetReqs()
			assert.Equal("/v1/video/details", reqs[1].URL.Path)
			assert.Len(reqs, 2)
			assert.Len(vals, 2)
		})

}

func TestStatus(t *testing.T) {
	assert := setupTest(t)
	serverStarted = time.Now()
	test_helper.RunSimpleGet("/v1/status",
		func(c *gin.Context) {
			status(c)
		},
		func(r *httptest.ResponseRecorder) {
			assert.Equal(200, r.Code)
			resp := test_helper.ParseJson(r.Body)
			assert.Equal(serverStarted.Format(time.RFC3339Nano), resp["server_started"])
			assert.Nil(resp["calls"])
		})
	testCalls := setupTestCalls()
	test_helper.RunSimpleGet("/v1/status",
		func(c *gin.Context) {
			status(c)
		},
		func(r *httptest.ResponseRecorder) {
			assert.Equal(200, r.Code)
			resp := test_helper.ParseJson(r.Body)
			assert.Equal(serverStarted.Format(time.RFC3339Nano), resp["server_started"])
			calls := resp["calls"].([]interface{})
			assert.Len(calls, len(testCalls))
		})
}

func TestSetupSynq(t *testing.T) {
	os.Setenv("SYNQ_API_KEY", "mykey")
	os.Setenv("SYNQ_API_URL", "myurl")
	assert := assert.New(t)
	setupSynq()
	assert.Equal("mykey", sApi.Key)
	assert.Equal("myurl", sApi.Url)
}

func TestSetupRouter(t *testing.T) {
	assert := assert.New(t)
	port := testLaunchRouter()
	resp, err := http.Post(fmt.Sprintf("http://localhost:%s/v1/details", port), "", strings.NewReader(""))
	assert.Nil(err)
	assert.Equal(400, resp.StatusCode)
	resp, err = http.Get(fmt.Sprintf("http://localhost:%s/v1/status", port))
	assert.Nil(err)
	assert.Equal(200, resp.StatusCode)
}

func TestParseDbUrl(t *testing.T) {
	assert := assert.New(t)
	name := parseDatabaseUrl("")
	assert.Equal("", name)
	name = parseDatabaseUrl("postgres://user:password@host.com:5432/dbname")
	assert.Equal("host=host.com port=5432 dbname=dbname user=user password=password", name)
	name = parseDatabaseUrl("postgres://user:password@host.com:5432/dbname?sslmode=disable")
	assert.Equal("host=host.com port=5432 dbname=dbname user=user password=password sslmode=disable", name)
	name = parseDatabaseUrl(DEFAULT_DB_URL)
	assert.Equal("host=localhost port=5432 dbname=gosample_test user=gotest password=gotest sslmode=disable", name)
}
