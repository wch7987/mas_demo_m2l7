package config

import "github.com/zeromicro/go-zero/rest"

type LLMConf struct {
	Model       string  `json:",optional"`
	ImageModel  string  `json:",optional"`
	APIKey      string  `json:",optional"`
	Region      string  `json:",default=cn"`
	BaseURL     string  `json:",optional"`
	Temperature float64 `json:",optional"`
	Timeout     int     `json:",default=600"`
	RetryCount  int     `json:",default=2"`
}

type XHSConf struct {
	DataOutputDir string `json:",default=./data/output"`
	ImageMaxSize  int    `json:",default=1024"`
	ImageQuality  int    `json:",default=85"`
	MaxImages     int    `json:",default=20"`
	MaxIter       int    `json:",default=3"`
}

type Config struct {
	rest.RestConf
	LLM LLMConf
	XHS XHSConf
}
