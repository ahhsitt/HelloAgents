// Package main 演示使用 HelloAgents 进行图像生成
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/ahhsitt/helloagents-go/pkg/image"
)

func main() {
	ctx := context.Background()

	// 从环境变量获取 API Key
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("请设置 OPENAI_API_KEY 环境变量")
		os.Exit(1)
	}

	// 创建 OpenAI 图像生成客户端
	provider, err := image.NewOpenAI(
		image.WithAPIKey(apiKey),
		image.WithModel(image.ModelDALLE3),
	)
	if err != nil {
		fmt.Printf("创建客户端失败: %v\n", err)
		os.Exit(1)
	}
	defer provider.Close()

	fmt.Printf("使用提供商: %s, 模型: %s\n", provider.Name(), provider.Model())
	fmt.Println("支持的尺寸:", provider.SupportedSizes())
	fmt.Println()

	// 生成图像
	fmt.Println("正在生成图像...")
	resp, err := provider.Generate(ctx, image.ImageRequest{
		Prompt:  "一只可爱的橘猫坐在窗台上，窗外是下雨的城市夜景，赛博朋克风格，霓虹灯倒影",
		Size:    image.ImageSize{Width: 1024, Height: 1024},
		Quality: image.QualityHD,
		Style:   image.StyleVivid,
	})
	if err != nil {
		fmt.Printf("生成失败: %v\n", err)
		os.Exit(1)
	}

	// 输出结果
	fmt.Println("生成成功!")
	fmt.Printf("生成时间: %d\n", resp.Created)
	for i, img := range resp.Images {
		fmt.Printf("\n图像 %d:\n", i+1)
		if img.URL != "" {
			fmt.Printf("  URL: %s\n", img.URL)
		}
		if img.RevisedPrompt != "" {
			fmt.Printf("  修改后的提示词: %s\n", img.RevisedPrompt)
		}
	}
}
