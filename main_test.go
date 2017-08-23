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
	gin.SetMode(gin.TestMode)
	go setupRouter(port)
	return port
}

func TestSave(t *testing.T) {
	assert := setupTest(t)
	call := ApiCall{}
	err := call.Save()
	assert.NotNil(err)
	assert.Equal("missing video id, can not save job", err.Error())
	call = ApiCall{VideoId: "123", Called: time.Now()}
	err = call.Save()
	assert.Nil(err)
	calls := []ApiCall{}
	e := db.Select(&calls, "select * from api_calls where true")
	assert.Nil(e)
	assert.Len(calls, 1)
	assert.Equal(call.VideoId, calls[0].VideoId)
	layout := "2006-01-02 03:04:05"
	assert.Equal("", calls[0].Type)
	assert.Equal(call.Called.UTC().Format(layout), calls[0].Called.UTC().Format(layout))
	call.Type = "details"
	call.Id = calls[0].Id
	err = call.Save()
	assert.Nil(err)
	calls = []ApiCall{}
	e = db.Select(&calls, "select * from api_calls where true")
	assert.Len(calls, 1)
	assert.Equal("details", calls[0].Type)
	// close the database and try to save
	db.Close()
	// ensure db is reset after this test
	defer setupDB()
	err = call.Save()
	assert.NotNil(err)
	assert.Equal("sql: database is closed", err.Error())
}

func TestCall(t *testing.T) {
	assert := setupTest(t)
	call := ApiCall{}
	err := call.Call()
	assert.NotNil(err)
	assert.Equal("Invalid uuid. Example: '1c0e3ea4529011e6991554a050defa20'.", err.Error())
	call.VideoId = test_helper.VIDEO_ID
	err = call.Call()
	assert.Nil(err)
	assert.Equal("/v1/video/details", call.Type)
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
	calls, err = getCalls(testCalls[0].VideoId)
	assert.Nil(err)
	assert.Len(calls, 1)
	assert.Equal(calls[0].VideoId, testCalls[0].VideoId)
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
	db.Close()
	// ensure db is reset after this test
	defer setupDB()
	test_helper.RunSimpleGet("/v1/status",
		func(c *gin.Context) {
			status(c)
		},
		func(r *httptest.ResponseRecorder) {
			assert.Equal(200, r.Code)
			resp := test_helper.ParseJson(r.Body)
			assert.Equal(serverStarted.Format(time.RFC3339Nano), resp["server_started"])
			assert.Nil(resp["calls"])
			assert.Equal("sql: database is closed", resp["error"].(string))
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

func TestGetPort(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(DEFAULT_PORT, getPort())
	os.Setenv("PORT", "1234")
	defer os.Unsetenv("PORT")
	assert.Equal("1234", getPort())
}

func TestParseDbUrl(t *testing.T) {
	assert := assert.New(t)
	name := parseDatabaseUrl("")
	assert.Equal("", name)
	name = parseDatabaseUrl("://abcd")
	assert.Equal("", name)
	name = parseDatabaseUrl("postgres://user:password@host.com:5432/dbname")
	assert.Equal("host=host.com port=5432 dbname=dbname user=user password=password", name)
	name = parseDatabaseUrl("postgres://user:password@host.com:5432/dbname?sslmode=disable")
	assert.Equal("host=host.com port=5432 dbname=dbname user=user password=password sslmode=disable", name)
	name = parseDatabaseUrl(DEFAULT_DB_URL)
	assert.Equal("host=localhost port=5432 dbname=gosample_test user=circleci password=circleci sslmode=disable", name)
}

func TestMain(t *testing.T) {
	assert := assert.New(t)
	go main()
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/v1/status", DEFAULT_PORT))
	assert.Nil(err)
	assert.Equal(200, resp.StatusCode)
}
