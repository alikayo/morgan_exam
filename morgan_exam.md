Candidate: **Jonathan D. Aujero**

Document Date: **January 21, 2022**

# Morgan IT Remote Technical Exam

- #### The web service project exam was written using GO programming language.

- #### You need to compile the program and when using linux you need to make the file executable.

- #### All errors are printed in the screen

  

### Program File: `morgan_exam` or `morgan_exam.exe`

#### Requires: `config.json`

- ##### The file must be in the same path with the program file.

  `config.json`

  ```
  {
    "ws_host": "localhost",
    "ws_port": 8080,
    "db_account": {
      "host": "localhost",
      "port": 5432,
      "username": "pgroot",
      "password": "pgsecret",
      "db_name": "dwmorgan"
    }
  }
  ```
  

  | Field               | Description                                                  |
  | ------------------- | ------------------------------------------------------------ |
  | ws_host             | Host IP address of the machine were the program will be running.  When blank it will bind to all IP address |
  | ws_port             | Port number in which the program will bind the HTTP server   |
  | db_account.host     | IP address of PostgreSQL database                            |
  | db_account.port     | Port number of PostgreSQL database                           |
  | db_account.username | Username of PostgreSQL login account                         |
  | db_account.password | Password of PostgreSQL login account                         |
  | db_account.db_name  | Database name to connect with                                |
  |                     |                                                              |




#### Table Name: `covid_observations`

- ##### This table must be created in Public Schema

```
create table covid_observations (
	s_no integer primary key not null,
	observation_date date not null,
	province_state varchar(200) not null,
	country_region varchar(200) not null,
	last_update timestamp not null,
	confirmed int not null default 0,
	deaths int not null default 0,
	recovered int not null default 0
)
create index covid_observations_idx01 on covid_observations(observation_date)
```



### How to Run in Linux

##### Run program with `-h` command parameter

- The program will display the available parameter usage

```
$ ./morgan_exam -h
Usage of ./morgan_exam:
  -load string
        parse and load covid observation csv file
        using relative path to program or absolute path:
         Ex.
         $ ./morgan_exam --load covid_19_data.csv
    
```
##### Start the program by loading a CSV file
```
$ ./morgan_exam --load covid_19_data.csv
```
>
>
>When there is an error during loading such as file not found or cannot established database connection the program will exit and prints the error message.
>
>

##### Start the program without loading a CSV file

```
$ ./morgan_exam
```


> 
>
> Running in Microsoft windows is similar to linux.  The only difference is that windows used different folder path separator and the generated program file have file extension `morgan_exam.exe`.  The program needs to be compiled using `GOOS=windows` environment variable.
>
>  

### Request

##### Method: **GET**
##### URI:  `http://domain:port/top/confirmed?observation_date=yyyy-mm-dd&max_results=3`

##### Query Parameters:
`observation_date`

- Any valid date format
- Required

`max_results`

- Max number of results returned
- Required




### Responses

##### Status: 200 - Ok

```
{
	"observation_date":"2020-01-22",
	"countries":[
		{"country":"Mainland China","confirmed":547,"deaths":17,"recovered":28},
        {"country":"Japan","confirmed":2,"deaths":0,"recovered":0},
        {"country":"Thailand","confirmed":2,"deaths":0,"recovered":0}
     ]
}
     
```


##### Status: 400 - Bad Request
```
observation_date or max_results parameters not found
```



##### Status: 400 - Bad Request
```
invalid observation date parameter
```



##### Status: 400 - Bad Request
```
invalid max result parameter
```



##### Status: 405 - Method not allowed
- if method used is not **GET**

  


##### Status 500 - Internal Server Error
```
error while retrieving the data
```



##### Status 500 - Internal Server Error
```
unable to serialize the result to json
```