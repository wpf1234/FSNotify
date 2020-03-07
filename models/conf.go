package models

type RedisConf struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	Pool     int    `json:"pool"`
	SetKey   string `json:"set_key"`
}

type LogConf struct {
	Path string `json:"path"`
	File string `json:"file"`
}

type SwitchConf struct {
	Path string `json:"path"`
}

type KafkaConf struct {
	Topic  string `json:"topic"`
	Broker string `json:"broker"`
	Key    string `json:"key"`
}

type MysqlConf struct {
	User string `json:"user"`
	Pwd  string `json:"pwd"`
	Host string `json:"host"`
	Db   string `json:"db"`
}