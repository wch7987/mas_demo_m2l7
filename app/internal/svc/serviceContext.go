package svc

import (
	"github.com/cloudwego/eino/components/model"
	"github.com/cms/mas_demo_m2l7/app/internal/config"
	imageutil "github.com/cms/mas_demo_m2l7/internal/image"
	"github.com/cms/mas_demo_m2l7/internal/llm"
	"github.com/cms/mas_demo_m2l7/internal/xhs"
)

type ServiceContext struct {
	Config  config.Config
	Service *xhs.Service
}

func NewServiceContext(c config.Config) *ServiceContext {
	llmCfg, err := llm.ConfigFromApp(
		c.LLM.Model,
		c.LLM.ImageModel,
		c.LLM.APIKey,
		c.LLM.Region,
		c.LLM.BaseURL,
		c.LLM.Temperature,
		c.LLM.Timeout,
		c.LLM.RetryCount,
	)
	if err != nil {
		panic("init llm: " + err.Error())
	}

	chatModel, err := llm.NewEinoChatModel(llmCfg)
	if err != nil {
		panic("init eino chat model: " + err.Error())
	}

	imageModel := chatModel
	if llmCfg.ImageModel != "" {
		imageModel = llm.WithModel(chatModel, llmCfg.ImageModel)
	}

	return NewServiceContextWith(c, chatModel, imageModel)
}

func NewServiceContextWith(c config.Config, textModel, imageModel model.ChatModel) *ServiceContext {
	maxIter := c.XHS.MaxIter
	if maxIter <= 0 {
		maxIter = 3
	}
	return &ServiceContext{
		Config: c,
		Service: &xhs.Service{
			Store: &imageutil.Store{
				OutputDir: c.XHS.DataOutputDir,
				MaxSize:   c.XHS.ImageMaxSize,
				Quality:   c.XHS.ImageQuality,
				MaxImages: c.XHS.MaxImages,
			},
			Runner: xhs.NewLiveRunner(textModel, imageModel, maxIter),
		},
	}
}
