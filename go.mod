module github.com/ChatDetectiveORG/payment-service

go 1.23.3

require (
	github.com/ChatDetectiveORG/shared v0.1.30
	github.com/go-pg/pg/v10 v10.15.0
	github.com/gomodule/redigo v1.9.3
	github.com/google/uuid v1.6.0
	github.com/rabbitmq/amqp091-go v1.10.0
	github.com/spf13/viper v1.21.0
	gopkg.in/telebot.v4 v4.0.0-beta.7
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-pg/zerochecker v0.2.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/rogpeppe/go-internal v1.10.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tmthrgd/go-hex v0.0.0-20190904060850-447a3041c3bc // indirect
	github.com/vmihailenco/bufpool v0.1.11 // indirect
	github.com/vmihailenco/msgpack/v5 v5.3.4 // indirect
	github.com/vmihailenco/tagparser v0.1.2 // indirect
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.41.0 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	mellium.im/sasl v0.3.1 // indirect
)

replace (
	github.com/ChatDetectiveORG/shared => ../shared
	gopkg.in/telebot.v4 => ../forks/telebot
)
