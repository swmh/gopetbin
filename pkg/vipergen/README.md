# Usage 

config.go
```go
//go:generate go run github.com/swmh/gopetbin/pkg/vipergen --path ../../config/ --name ".env" -env
//go:generate go run github.com/swmh/gopetbin/pkg/vipergen --path ../../config/ --name "config.yml" -yml
type Config struct {
	App struct {
		Addr              string `mapstructure:"addr"` 
		ReadTimeout       int    `mapstructure:"timeout_read"` /* in seconds */
		WriteTimeout      int    `mapstructure:"timeout_write"` /* in seconds */
		LogLevel          string `mapstructure:"log_level"`
	} `mapstructure:"app"` 

	DB struct {
		Addr string `mapstructure:"addr"`
		User string `mapstructure:"user"`
		Pass string `mapstructure:"pass"`
		Name string `mapstructure:"name"`
	} `mapstructure:"db"`

	Cache struct {
		Addr string `mapstructure:"addr"`
		User string `mapstructure:"user"`
		Pass string `mapstructure:"pass"`
		DB   int    `mapstructure:"db"`
	} `mapstructure:"cache"`
}
```

config.yml
```yaml
app:
  addr: ""
  timeout_read: 0 # in seconds
  timeout_write: 0 # in seconds
  log_level: ""
db:
  addr: ""
  user: ""
  pass: ""
  name: ""
cache:
  addr: ""
  user: ""
  pass: ""
  db: 0
```

.env
```env
APP_ADDR=string
APP_TIMEOUT_READ=0
APP_TIMEOUT_WRITE=0
APP_LOG_LEVEL=string

DB_ADDR=string
DB_USER=string
DB_PASS=string
DB_NAME=string

CACHE_ADDR=string
CACHE_USER=string
CACHE_PASS=string
CACHE_DB=0
```
