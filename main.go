package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

type Links struct {
	SourceLink string `json:"Source_link"`
	ShortLink  string `json:"Short_link"`
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

const sqlPath = "anton:qwerty123@tcp(127.0.0.1:3307)/links"

const port = ":8080"

func checkErr(err error, filelog *os.File) {
	errorLog := log.New(filelog, "ERROR\t", log.Ldate|log.Ltime)
	if err != nil {
		errorLog.Panic(err)
	}
}

func goodUrl(token string) bool {
	_, err := url.ParseRequestURI(token)
	if err != nil {
		return false
	}
	u, err := url.Parse(token)
	if err != nil || u.Scheme == "" || u.Path == "" {
		return false
	}
	return true
}

func linkShortening() string {
	shortlink := make([]byte, 5)
	rand.Seed(time.Now().Unix())

	for i := range shortlink {
		shortlink[i] = letterBytes[rand.Intn(len(letterBytes))]
	}

	return string(shortlink)
}

func repetitionСheck(link string, filelog *os.File) string {
	var cheklink string = ""
	tmpdb, err := sql.Open("mysql", sqlPath)
	checkErr(err, filelog)
	defer tmpdb.Close()

	row := tmpdb.QueryRow("select Short_link from links_data where Source_link = ?", link)

	if err = row.Scan(&cheklink); err != nil {
		return cheklink
	}

	return cheklink
}

func mainPage(w http.ResponseWriter, r *http.Request) {

	flog, openErr := os.OpenFile("log/fLog.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if openErr != nil {
		panic(openErr)
	}
	infoLog := log.New(flog, "INFO\t", log.Ldate|log.Ltime)
	defer flog.Close()

	infoLog.Printf("go to \"localhost:%s\" address", port)
	tmp, err := template.ParseFiles("static/index.html")
	checkErr(err, flog)

	err = tmp.Execute(w, nil)
	checkErr(err, flog)
}

func addLinkToShortining(w http.ResponseWriter, r *http.Request) {

	flog, openErr := os.OpenFile("log/fLog.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if openErr != nil {
		panic(openErr)
	}
	infoLog := log.New(flog, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(flog, "ERROR\t", log.Ldate|log.Ltime)
	defer flog.Close()

	infoLog.Printf("go to \"localhost:%s/shortering/%s\" address", port, r.URL.Path[12:])

	if !goodUrl(r.URL.Path[12:]) {
		errorLog.Printf("wrong URL")
		return
	}

	tmpl, err := template.ParseFiles("static/toshort.html")
	checkErr(err, flog)

	tmpdb, err := sql.Open("mysql", sqlPath)
	checkErr(err, flog)
	defer tmpdb.Close()

	err = tmpdb.Ping()
	checkErr(err, flog)

	conn, err := sqlx.Connect("mysql", sqlPath)
	checkErr(err, flog)

	link := Links{}

	link.SourceLink = r.URL.Path[12:]

	checksourcelink := repetitionСheck(link.SourceLink, flog)

	if checksourcelink != "" {
		link.ShortLink = checksourcelink
	} else {
		tmpstr := strings.Split(link.SourceLink, "/")
		link.ShortLink = tmpstr[0] + "/" + tmpstr[1] + "/" + linkShortening()
		_, err = conn.Exec("INSERT INTO links_data (`Source_link`, `Short_link`) VALUES(?, ?)", link.SourceLink, link.ShortLink)

		checkErr(err, flog)

	}

	err = tmpl.Execute(w, link)
	checkErr(err, flog)
}

func linksPage(w http.ResponseWriter, r *http.Request) {

	flog, openErr := os.OpenFile("log/fLog.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if openErr != nil {
		panic(openErr)
	}
	infoLog := log.New(flog, "INFO\t", log.Ldate|log.Ltime)
	defer flog.Close()

	infoLog.Printf("go to \"localhost:%s/links\" address", port)

	links := []Links{}
	link := Links{}

	tmpdb, err := sql.Open("mysql", sqlPath)
	checkErr(err, flog)
	defer tmpdb.Close()

	rows, _ := tmpdb.Query("select * from links_data")

	for rows.Next() {
		err = rows.Scan(&link.SourceLink, &link.ShortLink)
		checkErr(err, flog)
		links = append(links, link)
	}

	js, err := json.Marshal(links)
	checkErr(err, flog)
	w.Write(js)
}

func redirect(w http.ResponseWriter, r *http.Request) {

	flog, openErr := os.OpenFile("log/fLog.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if openErr != nil {
		panic(openErr)
	}
	infoLog := log.New(flog, "INFO\t", log.Ldate|log.Ltime)
	defer flog.Close()

	infoLog.Printf("go to \"localhost:%s/redirectTo/%s\" address", port, r.URL.Path[12:])

	link := Links{}
	link.ShortLink = r.URL.Path[12:]

	tmpdb, err := sql.Open("mysql", sqlPath)
	checkErr(err, flog)
	defer tmpdb.Close()

	row := tmpdb.QueryRow("select Source_link from links_data where Short_link = ?", link.ShortLink)

	err = row.Scan(&link.SourceLink)
	checkErr(err, flog)
	fmt.Fprintf(w, "<script>location='%s';</script>", link.SourceLink)
}

func main() {

	flog, openErr := os.OpenFile("log/fLog.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0777)
	if openErr != nil {
		panic(openErr)
	}
	defer flog.Close()

	infoLog := log.New(flog, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(flog, "ERROR\t", log.Ldate|log.Ltime)

	http.HandleFunc("/", mainPage)
	http.HandleFunc("/shortering/", addLinkToShortining)
	http.HandleFunc("/redirectTo/", redirect)
	http.HandleFunc("/links/", linksPage)
	infoLog.Printf("Запуск сервера на %s", port)
	errorLog.Fatal(http.ListenAndServe(port, nil))

}
