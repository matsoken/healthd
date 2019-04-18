package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"

	"gopkg.in/yaml.v2"
)

// Checker : holds a single health check's configuration information
type Checker struct {
	Name  string
	Type  string
	Props map[string]string
}

//CheckResult : holds the results of the health test
type CheckResult struct {
	Name        string
	Status      bool
	Message     string
	Elapsedtime int64
}

type jsonOut struct {
	Status bool
	Checks []CheckResult
}

var tmpl = `<!doctype html>
<html lang="en">
  <head>
    <!-- Required meta tags -->
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">

    <!-- Bootstrap CSS -->
    <link rel="stylesheet" href="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/css/bootstrap.min.css" integrity="sha384-ggOyR0iXCbMQv3Xipma34MD+dH/1fQ784/j6cY/iJTQUOhcWr7x9JvoRxT2MZw1T" crossorigin="anonymous">

    <title>HealthD</title>
  </head>
  <body>


	<div class="container">
            <h3>Health</h3>
			<div class="row">
				<div class="col-sm-1 text-white bg-secondary">
					 
	  			</div>
                <div class="col-sm-2 text-white bg-secondary">
                      Name 
                </div>
                <div class="col-sm-1 text-white bg-secondary">
                     Status
                </div>
                <div class="col-sm-1 text-white bg-secondary">
                    Resp(ms)
                </div>
               <div class="col-sm-4 text-white bg-secondary">
                  Message
                </div>
               
              </div>

		{{range . }}
			<div class="row">
			  <div class="col-sm-1{{if not .Status}} bg-danger{{else}} bg-success{{end}}">
					
	 		  </div>
              <div class="col-sm-2">
                    {{.Name}} 
              </div>
              <div class="col-sm-1">
					{{.Status}}
					
              </div>
              <div class="col-sm-1">
                    {{.Elapsedtime}}
              </div>
			  <div class="col-sm-4">
			  	{{.Message}}
			  </div>

              
             
            </div>          
        
        
		{{end}}
    </div>
	
    <!-- Optional JavaScript -->
    <!-- jQuery first, then Popper.js, then Bootstrap JS -->
    <script src="https://code.jquery.com/jquery-3.3.1.slim.min.js" integrity="sha384-q8i/X+965DzO0rT7abK41JStQIAqVgRVzpbzo5smXKp4YfRvH+8abtTE1Pi6jizo" crossorigin="anonymous"></script>
    <script src="https://cdnjs.cloudflare.com/ajax/libs/popper.js/1.14.7/umd/popper.min.js" integrity="sha384-UO2eT0CpHqdSJQ6hJty5KVphtPhzWj9WO1clHTMGa3JDZwrnQq4sF86dIHNDz0W1" crossorigin="anonymous"></script>
    <script src="https://stackpath.bootstrapcdn.com/bootstrap/4.3.1/js/bootstrap.min.js" integrity="sha384-JjSmVgyd0p3pXB1rRibZUAYoIIy6OrQ6VrjIEaFf/nJGzIxFDsf4x0xIM+B07jRM" crossorigin="anonymous"></script>
  </body>
</html>
`
var checkArray []Checker //This holds an array of all the health checks that need to be performed

func main() {

	var configfile string
	var listen string
	flag.StringVar(&configfile, "config", "config.yml", "Config file to use")
	flag.StringVar(&listen, "listen", ":9180", "Listen on address:port")
	flag.Parse()

	checkArray = loadConfig(configfile)

	r := mux.NewRouter()

	r.HandleFunc("/", uihandler)
	r.HandleFunc("/health", handler)
	r.HandleFunc("/health/{check}", singlehandler)

	log.Printf("Starting Listener on %s\n", listen)
	http.ListenAndServe(listen, r)
}

func runChecks(checks []Checker) []CheckResult {
	var results []CheckResult

	for i, check := range checks {
		var status bool
		var statusmsg string
		log.Printf("%d %s\n", i, check.Name)
		start := time.Now()
		switch check.Type {
		case "DB":
			status, statusmsg = runDBCheck(check)
		case "HTTP":
			status, statusmsg = runHTTPCheck(check)
		case "TCP":
			status, statusmsg = runTCPCheck(check)
		default:
			fmt.Printf("Unknown type %s, skipping\n", check.Type)
		}
		elapsed := time.Since(start)

		txt := fmt.Sprintf("%s %s Check %s Status: %t", check.Name, check.Type, statusmsg, status)
		thisresult := CheckResult{check.Name, status, txt, (elapsed.Nanoseconds() / 1000000)}
		results = append(results, thisresult)

	}
	return results
}

func loadConfig(configfile string) []Checker {
	var cfg []Checker

	yamlFile, err := ioutil.ReadFile(configfile)
	if err != nil {
		log.Printf("Could not load %s #%v ", configfile, err)
		panic("Quitting")
	}
	yerr := yaml.Unmarshal(yamlFile, &cfg)
	if yerr != nil {
		log.Fatalf("Unmarshal: %v", yerr)
	}

	return cfg
}

func handler(w http.ResponseWriter, r *http.Request) {

	log.Printf("%s %s", r.RemoteAddr, r.RequestURI)
	results := runChecks(checkArray)
	status := getOverallStatus(results)

	jsonout := jsonOut{status, results}

	jData, err := json.MarshalIndent(jsonout, "", "    ")
	if err != nil {
		// handle error
		panic("ouch json")
	}
	if status == false {
		w.WriteHeader(500)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jData)
}

func uihandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.RemoteAddr, r.RequestURI)
	results := runChecks(checkArray)
	t := template.New("main") //name of the template is main
	t, _ = t.Parse(tmpl)      // parsing of template string
	t.Execute(w, results)

}

func singlehandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var singlecheck []Checker
	log.Printf("%s %s %s", r.RemoteAddr, r.RequestURI, vars["check"])
	for _, v := range checkArray {
		if v.Name == vars["check"] {
			singlecheck = append(singlecheck, v)
		}
	}
	if len(singlecheck) < 1 {
		w.WriteHeader(500)
		w.Write([]byte("No such check defined"))
		return
	}
	results := runChecks(singlecheck)
	status := getOverallStatus(results)

	jsonout := jsonOut{status, results}

	jData, err := json.MarshalIndent(jsonout, "", "    ")
	if err != nil {
		// handle error
		panic("ouch json")
	}
	if status == false {
		w.WriteHeader(500)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(jData)
}

func getOverallStatus(checks []CheckResult) bool {
	var status bool
	status = true
	for _, v := range checks {
		status = status && v.Status
	}
	return status

}

func validateProps(requiredProps []string, props map[string]string) error {

	for _, v := range requiredProps {
		if _, ok := props[v]; ok {
		} else {
			return fmt.Errorf("Invalid config missing %s", v)
		}
	}

	return nil
}

func runDBCheck(chk Checker) (bool, string) {

	//Validate all the expected DB props exist
	err := validateDBProps(chk)
	if err != nil {
		return false, err.Error()
	}
	//Run actualDBCheck
	err = connectDB(chk.Props["dbdriver"], chk.Props["connstr"], chk.Props["query"])
	if err != nil {
		return false, err.Error()
	}
	return true, "ok"
}

func runHTTPCheck(chk Checker) (bool, string) {

	//Validate all the expected  props exist
	err := validateHTTPProps(chk)
	if err != nil {
		return false, err.Error()
	}
	//Run actual Check
	err = connectHTTP(chk.Props["url"], chk.Props["method"])
	if err != nil {
		return false, err.Error()
	}
	return true, "ok"
}

func runTCPCheck(chk Checker) (bool, string) {

	//Validate all the expected  props exist
	err := validateTCPProps(chk)
	if err != nil {
		return false, err.Error()
	}
	//Run actual Check
	err = connectTCP(chk.Props["addr"])
	if err != nil {
		return false, err.Error()
	}
	return true, "ok"
}

func validateDBProps(chk Checker) error {

	props := []string{"dbdriver", "connstr", "query"}
	err := validateProps(props, chk.Props)

	return err

}

func connectDB(dbdriver string, connstr string, query string) error {

	db, err := sql.Open(dbdriver, connstr)
	if err != nil {
		return err
	}
	rows, err := db.Query(query)
	if err != nil {
		return err
	}
	rows.Close()

	defer db.Close()
	return nil
}

func validateHTTPProps(chk Checker) error {
	props := []string{"url", "method"}
	err := validateProps(props, chk.Props)

	return err

}

func connectHTTP(url string, method string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(ioutil.Discard, resp.Body)
	stat := resp.StatusCode
	if stat > 399 {
		return fmt.Errorf("HTTP Response Code is %d", stat)
	}
	return nil

}

func validateTCPProps(chk Checker) error {
	props := []string{"addr"}
	err := validateProps(props, chk.Props)

	return err

}

func connectTCP(addr string) error {
	ts := time.Second * 5
	conn, err := net.DialTimeout("tcp", addr, ts)
	if err != nil {
		return err
	}
	defer conn.Close()

	return nil

}
