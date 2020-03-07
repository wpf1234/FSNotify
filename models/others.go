package models

type SwitchWarn struct {
	Port       string `json:"port"`
	AccessVlan int    `json:"accessVlan"`
	AuthVlan   int    `json:"authVlan"`
}

type Warning struct {
	Category int    `json:"category"`
	Level    int    `json:"level"`
	Tm       int64  `json:"tm"` // time.Now().UnixNano /1e6
	Ip       string `json:"ip"`
	Host     string `json:"host"`
	Event    int    `json:"event"`
	Title    string `json:"title"`
	Msg      string `json:"msg"`
	Origin   string `json:"origin"`
	Ext      Ext    `json:"ext"`
}

type Ext struct {
	//Source          string      `json:"source"`
	//Address          string      `json:"address"`
	//Port             interface{} `json:"port"`
	//SoftwareCategory string      `json:"softwareCategory"`
	ConfVlans []SwitchWarn `json:"confVlans"`
}

type SwitchVlan struct {
	Port string      `json:"port"`
	Vlan interface{} `json:"vlan"`
}
