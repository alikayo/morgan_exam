package main

import (
	"database/sql"
	_ "database/sql/driver"
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/lib/pq"
)

type DbConn struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	DbName   string `json:"db_name"`
}

type Config struct {
	WSHost    string `json:"ws_host"`
	WSPort    int    `json:"ws_port"`
	DbAccount DbConn `json:"db_account"`
}
type CovidData struct {
	SNo             int       `json:"-"`
	ObservationDate time.Time `json:"-"`
	ProvinceState   string    `json:"-"`
	CountryRegion   string    `json:"country"` // gorm:"country_region"
	LastUpdate      time.Time `json:"-"`
	Confirmed       float64   `json:"confirmed"`
	Deaths          float64   `json:"deaths"`
	Recovered       float64   `json:"recovered"`
}

type Result struct {
	ObservationDate string      `json:"observation_date"`
	Countries       []CovidData `json:"countries"`
}

type Controller struct {
	Cn *sql.DB
}

func main() {
	pwd := filepath.Dir(os.Args[0])
	progname := filepath.Base(os.Args[0])
	data_file := flag.String("load", "", fmt.Sprintf("parse and load covid observation csv file\nusing relative path to program or absolute path:\n Ex.\n $ ./%s --load covid_19_data.csv\n", progname))

	flag.Parse()

	cfg, err := getConfig(filepath.Join(pwd, "config.json"))

	if err != nil {
		log.Fatalln(err)
	}
	cn, err2 := connectPg(&cfg.DbAccount)
	if err2 != nil {
		log.Fatalln(err2)
	}

	if *data_file != "" {
		parseAndLoad(cn, *data_file)
	}
	if len(os.Args) > 1 && *data_file == "" {
		log.Println("invalid parameters:", os.Args[1:])
		os.Exit(0)
	}
	//GET /top/confirmed?observation_date=yyyy-mm-dd&max_results=2
	controller := Controller{Cn: cn}
	http.HandleFunc("/top/confirmed", controller.topConfirmedCovid)
	fmt.Printf("listening on %s:%d\n", cfg.WSHost, cfg.WSPort)
	host := cfg.WSHost
	if cfg.WSHost == "" {
		host = "localhost"
	}
	fmt.Printf("Example:\n\tGET: http://%s:%d/top/confirmed?observation_date=yyyy-mm-dd&max_results=2\n", host, cfg.WSPort)
	http.ListenAndServe(fmt.Sprintf("%s:%d", cfg.WSHost, cfg.WSPort), nil)
}

func getConfig(conf string) (*Config, error) {
	log.Printf("loading config file `%s`", conf)
	dat, err := ioutil.ReadFile(conf)
	if err != nil {
		return nil, err
	}
	cfg := new(Config)
	err = json.Unmarshal(dat, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func connectPg(p *DbConn) (*sql.DB, error) {
	db, err := sql.Open("postgres", fmt.Sprintf("host=%s port=%v user=%s password=%s dbname=%s sslmode=disable", p.Host, p.Port, p.Username, p.Password, p.DbName))
	if err != nil {
		return nil, err
	}
	err = db.Ping()
	return db, err
}

func parseAndLoad(db *sql.DB, dfile string) {
	fmt.Printf("Loading data file: `%s`\n", dfile)

	csvfile, err := os.Open(dfile)
	if err != nil {
		log.Fatalln(err)
	}
	defer csvfile.Close()

	r := csv.NewReader(csvfile)
	records, err := r.ReadAll()
	if err != nil {
		log.Fatalln(err)
	}

	tx, _ := db.Begin()
	insert_stmt, _ := tx.Prepare(pq.CopyIn("covid_observations", "s_no", "observation_date", "province_state", "country_region", "last_update", "confirmed", "deaths", "recovered"))
	// skip header start at row 2
	for i := 1; i < len(records); i++ {
		rec := records[i]
		data, err2 := toCovidData(rec)

		if err2 != nil {
			log.Printf("line %d: %s\n", i, err2)
			continue
		}
		_, err2 = insert_stmt.Exec(data.SNo, data.ObservationDate, data.ProvinceState, data.CountryRegion, data.LastUpdate, data.Confirmed, data.Deaths, data.Recovered)
		if err2 != nil {
			if !strings.Contains(err2.Error(), "duplicate") {
				log.Println(err2)
			}
		}
	}
	// flush

	_, err = insert_stmt.Exec()
	if err != nil {
		if !strings.Contains(err.Error(), "duplicate") {
			log.Fatalln(err)
		}
		tx.Rollback()
		return
	}

	// commit to table
	err = tx.Commit()
	if err != nil {
		log.Fatalln(err)
	}

}

func toCovidData(row []string) (*CovidData, error) {
	if len(row) != 8 {
		return nil, errors.New("invalid record")
	}
	var err error
	d := new(CovidData)
	d.SNo, err = strconv.Atoi(row[0])
	if err != nil {
		return nil, errors.New("invalid SNo column value")
	}
	d.ObservationDate, err = DateTimeParse(row[1])
	if err != nil {
		return nil, errors.New("invalid ObservationDate column value")
	}
	if len(row[2]) > 200 {
		return nil, errors.New("column Province/State exceeds maximum length of 200 characters")
	}
	d.ProvinceState = row[2]

	if len(row[3]) > 200 {
		return nil, errors.New("column Country/Region exceeds maximum length of 200 characters")
	}
	d.CountryRegion = row[3]

	d.LastUpdate, err = DateTimeParse(row[4])
	if err != nil {
		return nil, errors.New("invalid ObservationDate column value")
	}
	d.Confirmed, err = strconv.ParseFloat(row[5], 32)
	if err != nil {
		return nil, errors.New("invalid Confirmed column value")
	}
	d.Deaths, err = strconv.ParseFloat(row[6], 32)
	if err != nil {
		return nil, errors.New("invalid Deaths column value")
	}
	d.Recovered, err = strconv.ParseFloat(row[7], 32)
	if err != nil {
		return nil, errors.New("invalid Recovered column value")
	}
	return d, nil
}

func (me *Controller) topConfirmedCovid(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	qry := req.URL.Query()

	// check/read query parameters
	observation_date, ok := qry["observation_date"]
	max_results, ok2 := qry["max_results"]
	if !ok || !ok2 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("observation_date or max_results parameter not found"))
		return
	}

	// evaluate to parameters
	d, err := DateTimeParse(observation_date[0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid observation date parameter"))
		return
	}
	max_row, err2 := strconv.Atoi(max_results[0])
	if err2 != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("invalid max result parameter"))
		return
	}

	// get covid data
	result, err2 := me.getConfirmedData(d, max_row)
	if err2 != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error while retrieving the data"))
		return
	}
	byts, err3 := result.ToJSONBytes()
	if err3 != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("unable to serialize the result to json"))
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Write(byts)

}

func (me *Controller) getConfirmedData(observation_date time.Time, max_row int) (*Result, error) {
	recordset, err := me.Cn.Query("SELECT country_region, sum(confirmed) c, sum(deaths) d, sum(recovered) r FROM covid_observations WHERE observation_date=$1 GROUP BY country_region ORDER BY c DESC LIMIT $2", observation_date, max_row)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	res := new(Result)
	res.ObservationDate = observation_date.Format("2006-01-02")
	res.Countries = make([]CovidData, 0)
	for recordset.Next() {
		row := new(CovidData)
		err = recordset.Scan(&row.CountryRegion, &row.Confirmed, &row.Deaths, &row.Recovered)
		if err != nil {
			return nil, err
		}
		res.Countries = append(res.Countries, *row)
	}
	return res, nil
}

func (me *Result) ToJSONBytes() ([]byte, error) {
	sJSON, err := json.Marshal(me)

	return sJSON, err
}
