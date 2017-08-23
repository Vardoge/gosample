package main

import (
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/SYNQfm/SYNQ-Golang/synq"
	"github.com/gin-gonic/gin"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var db *sqlx.DB
var dbaddr string
var sApi synq.Api
var serverStarted time.Time

const (
	DEFAULT_PORT   = "41414"
	DEFAULT_DB_URL = "postgres://circleci:circleci@localhost:5432/gosample_test?sslmode=disable"
)

type ApiCall struct {
	Id        int64         `db:"id" json:"id"`
	Type      string        `db:"type" json:"type"`
	CreatedAt time.Time     `db:"ctime" json:"-"`
	Called    time.Time     `db:"called" json:"called"`
	VideoId   string        `db:"video_id" json:"video_id"`
	Taken     time.Duration `db:"taken" json:"taken"`
	Error     string        `db:"error" json:"error"`
	Video     synq.Video    `db:"result" json:"video"`
}

func init() {
	setupSynq()
	setupDB()
}

func setupSynq() {
	key := os.Getenv("SYNQ_API_KEY")
	if key == "" {
		log.Println("WARNING : no Synq API key specified")
	}
	sApi = synq.New(key)
	url := os.Getenv("SYNQ_API_URL")
	if url != "" {
		sApi.Url = url
	}
}

func setupDB() {
	db_url := os.Getenv("DATABASE_URL")
	if db_url == "" {
		db_url = DEFAULT_DB_URL
	}
	dbaddr = parseDatabaseUrl(db_url)
	db = sqlx.MustConnect("postgres", dbaddr)
}

// this parses the database url and returns it in the format sqlx.DB expects
func parseDatabaseUrl(dbUrl string) string {
	if dbUrl == "" {
		return ""
	}
	u, e := url.Parse(dbUrl)
	if e != nil {
		log.Printf("Error parsing '%s' : %s\n", dbUrl, e.Error())
		return ""
	}
	str := fmt.Sprintf("host=%s port=%s dbname=%s",
		u.Hostname(), u.Port(), strings.Replace(u.Path, "/", "", -1))
	if u.User != nil && u.User.Username() != "" {
		pass, set := u.User.Password()
		str = str + " user=" + u.User.Username()
		if set {
			str = str + " password=" + pass
		}
	}
	ssl := u.Query().Get("sslmode")
	if ssl != "" {
		str = str + " sslmode=" + ssl
	}
	return str
}

func (a *ApiCall) Exists() bool {
	return a.Id > 0
}

func (a *ApiCall) Save() error {
	if a.VideoId == "" {
		return errors.New("missing video id, can not save job")
	}

	var query string
	args := []interface{}{a.VideoId, a.Called, a.Taken, a.Type, a.Error, a.Video}
	if a.Exists() {
		query = `UPDATE api_calls SET video_id = $1, called = $2, taken = $3, type = $4, error = $5, result = $6 WHERE id = $7`
		args = append(args, a.Id)
	} else {
		query = `INSERT INTO api_calls (video_id, called, taken, type, error, result) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`
	}
	_, err := db.Exec(query, args...)
	if err != nil {
		return err
	}
	return nil

}

func (a *ApiCall) Call() error {
	defer a.Save()
	start := time.Now()
	video, err := sApi.GetVideo(a.VideoId)
	a.Taken = time.Since(start)
	if err != nil {
		a.Error = err.Error()
		return err
	}
	a.Type = "/v1/video/details"
	a.VideoId = video.Id
	a.Video = video
	return nil
}

// This will just get the video id and call the Synq api for the details
func details(c *gin.Context) {
	a := ApiCall{}
	c.BindJSON(&a)

	if a.VideoId == "" {
		c.JSON(400, gin.H{
			"message": "missing 'video_id'",
		})
		return
	}
	err := a.Call()
	if err != nil {
		c.JSON(400, gin.H{
			"message": "failled to make api call : " + err.Error(),
		})
		return

	}
	c.JSON(200, gin.H{
		"message": a.Video,
		"id":      a.VideoId,
	})
}

// Return status of transcode service or the results of a specific video or job
func status(c *gin.Context) {
	data := make(map[string]interface{})
	data["server_started"] = serverStarted
	calls, err := getCalls()
	if err != nil {
		data["error"] = err.Error()
	}
	data["calls"] = calls
	c.JSON(200, data)
}

func getCalls(ids ...string) (calls []ApiCall, err error) {
	where := ""
	args := []interface{}{}
	if len(ids) > 0 {
		where = "where video_id = $1"
		args = append(args, ids[0])
	} else {
		where = "where TRUE"
	}
	query := "SELECT * FROM api_calls " + where
	err = db.Select(&calls, query, args...)
	return calls, err
}

func getPort() string {
	port := os.Getenv("PORT")
	if port == "" {
		port = DEFAULT_PORT
	}
	return port
}

func setupRouter(port string) {
	router := gin.Default()
	v1 := router.Group("/v1")
	{
		v1.POST("/details", details)
		v1.GET("/status", status)
	}
	log.Println("Running server on port :", port)
	serverStarted = time.Now()
	router.Run(":" + port)
}

func main() {
	setupRouter(getPort())
}
