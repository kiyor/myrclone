package main

type Stats struct {
	Bytes        uint64          `json:"bytes"`
	Checks       int             `json:"checks"`
	Deletes      int             `json:"deletes"`
	ElapsedTime  float64         `json:"elapsedTime"`
	Errors       int             `json:"errors"`
	FatalError   bool            `json:"fatalError"`
	RetryError   bool            `json:"retryError"`
	Speed        float64         `json:"speed"`
	Transferring []*Transferring `json:"transferring"`
	Transfers    int             `json:"transfers"`
}

type Transferring struct {
	Bytes      int64   `json:"bytes"`
	Eta        int     `json:"eta"`
	Group      string  `json:"group"`
	Name       string  `json:"name"`
	Percentage int     `json:"percentage"`
	Size       int64   `json:"size"`
	Speed      float64 `json:"speed"`
	SpeedAvg   float64 `json:"speedAvg"`
}
