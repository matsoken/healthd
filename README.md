# healthd
Health check web service daemon to validate external service and db dependencies written in GO

This web service is intended to be a simple to deploy heath checker to validate the status of a systems dependencies.  The intent is to decrease the time it takes to focus troubelshooting on the real issue. It provides both a human readable output and a JSON consumable output for consumption by monitoring sytems and scripts. The service is meant to be run on the web server, docker host, application server that consumes the services so that health is an accurate reflection from the hosts point of view.

Only 2 files to deploy, healthd binary and yaml config file.

Currently supports HTTP, TCP and SQL DB checks (MSSQL and MySQL).

* Human output: /
* JSON: /health
* JSON Filtered:  /health/checkname

Example output human readable http://server:9180, green is up, red is down:
![screesnhot](https://raw.githubusercontent.com/matsoken/healthd/assets/healthd-user.PNG)

Same output in JSON http://server:9180/health, note the rollup status:
```json
curl http://localhost:9180/health
{
    "Status": false,
    "Checks": [
        {
            "Name": "WeatherAPI",
            "Status": true,
            "Message": "WeatherAPI HTTP Check ok Status: true",
            "Elapsedtime": 134
        },
        {
            "Name": "GitHub SSH",
            "Status": true,
            "Message": "GitHub SSH TCP Check ok Status: true",
            "Elapsedtime": 100
        },
        {
            "Name": "MySQL Check",
            "Status": false,
            "Message": "MySQL Check DB Check Error 1045: Access denied for user 'user'@'localhost' (using password: YES) Status: false",
            "Elapsedtime": 2
        }
    ]
}

```

Same JSON output filtered for a single service http://server:9180/health/WeatherAPI, NOTE: status is for filtered service only:
```
curl http://localhost:9180/health/WeatherAPI
{
    "Status": true,
    "Checks": [
        {
            "Name": "WeatherAPI",
            "Status": true,
            "Message": "WeatherAPI HTTP Check ok Status: true",
            "Elapsedtime": 487
        }
    ]
}
```
### Quick Setup:

Get and build the app:
```
mkdir healthd
cd healthd
git clone https://github.com/matsoken/healthd
go build
```

Or simply download and extract a release.
https://github.com/matsoken/healthd/releases

Edit the config.yml and add health checks, include all external dependencies that your application/system/process needs:
```yaml
- name: PricingServiceCheck
  type: HTTP
  props:
    method: GET
    url: https://pricingserver/pricing/health
- name: ProprietaryServiceCheck
  type: TCP
  props:
    addr: internalserver:22
- name: PrimaryDatabaseMySQL
  type: DB
  props:
    connstr: user:pwd@/test # Use connection strings supported by driver
    query: select 1
    dbdriver: mysql
```

Execute the application, by default it loads config.yml from the working directory:

```
./healthd
2019/04/14 13:06:34 Starting Listener on :9180
```

or
```
.\healthd.exe
2019/04/14 13:06:34 Starting Listener on :9180
```

Navigate to http://host:9180

Command Line Args:
```
Usage healthd.exe:
  -config string
        Config file to use (default "config.yml")
  -listen string
        Listen on address:port (default ":9180")
```

### Monitoring Systems and Scripts

For granularity of alerting, both group and single check status endpoints are available

The application will return HTTP 500 for a status that is not true.  i.e http://server:9180/health will return 200 if all services are healthy and 500 if any single service is unhealthy.  Convenient for an overall health status monitor.

A single service monitor will report 200 for a status = true and 500 for a status = false on the one service. e.g. http://server:9180/health/WeatherAPI.








