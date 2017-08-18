package main

import (
	"fmt"
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
	calls = append(calls, ApiCall{VideoId: "123"})
	calls = append(calls, ApiCall{VideoId: "456"})
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

}

func TestGetCalls(t *testing.T) {
	assert := setupTest(t)
	calls, err := getCalls()
	assert.Nil(err)
	assert.Len(calls, 0)
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
			assert.Equal("", resp["message"])
		})
	sApi.Key = key
	test_helper.RunSimplePost("/v1/details", `{"video_id" : "`+test_helper.VIDEO_ID+`"}`,
		func(c *gin.Context) {
			details(c)
		},
		func(r *httptest.ResponseRecorder) {
			assert.Equal(200, r.Code)
			resp := test_helper.ParseJson(r.Body)
			assert.Equal("", resp["message"])
			reqs, vals := test_helper.GetReqs()
			assert.Len(reqs, 1)
			assert.Len(vals, 1)
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
			calls := resp["calls"].([]interface{})
			assert.Len(calls, 0)
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
	resp, err := http.Post(fmt.Sprintf("http://localhost:%s/v1/transcode", port), "", strings.NewReader(""))
	assert.Nil(err)
	// should fail on StatusCode 400
	assert.Equal(400, resp.StatusCode)
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
	assert.Equal("host=localhost port=5432 dbname=hydra_test user=hydra password=hydra sslmode=disable", name)
}

func TestMain(t *testing.T) {
	assert := setupTest(t)
	defer func() {
		// recover from panic if one occured. Set err to nil otherwise.
		if recover() == nil {
			assert.Fail("did not raise panic")
		}
	}()
	main()
	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/v1/status", DEFAULT_PORT))
	assert.Nil(err)
	// should fail on StatusCode 400
	assert.Equal(200, resp.StatusCode)
}
