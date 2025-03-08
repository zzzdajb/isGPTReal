package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/user/isGPTReal/internal/api"
	"github.com/user/isGPTReal/internal/detector"
)

func main() {
	// 命令行参数
	port := flag.Int("port", 8080, "HTTP服务器端口")
	endpoint := flag.String("endpoint", "", "OpenAI兼容API端点")
	apiKey := flag.String("apikey", "", "API密钥")
	model := flag.String("model", "gpt-4o-mini", "要使用的模型")
	interval := flag.Int("interval", 0, "检测间隔（分钟），0表示不自动检测")
	flag.Parse()

	// 验证必要的参数
	if *endpoint == "" {
		*endpoint = os.Getenv("OPENAI_ENDPOINT")
		if *endpoint == "" {
			log.Fatal("必须指定API端点 (--endpoint 或环境变量 OPENAI_ENDPOINT)")
		}
	}

	if *apiKey == "" {
		*apiKey = os.Getenv("OPENAI_API_KEY")
		if *apiKey == "" {
			log.Fatal("必须指定API密钥 (--apikey 或环境变量 OPENAI_API_KEY)")
		}
	}

	// 创建配置
	config := detector.Config{
		Endpoint:    *endpoint,
		APIKey:      *apiKey,
		Model:       *model,
		Interval:    *interval,
		MaxHistory:  100,
		SaveRawResp: true,
	}

	// 创建服务器
	server := api.NewServer(config)

	// 启动服务器
	addr := fmt.Sprintf(":%d", *port)
	log.Printf("启动服务器，监听端口 %d...\n", *port)
	log.Printf("你可以访问 http://localhost:%d 查看结果", *port)
	if err := server.Run(addr); err != nil {
		log.Fatal(err)
	}
}
