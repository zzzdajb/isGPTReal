package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/user/isGPTReal/internal/api"
	"github.com/user/isGPTReal/internal/detector"
)

// 默认设置常量
const (
	DefaultPort       = 8080
	DefaultModel      = "gpt-4o-mini"
	DefaultInterval   = 0
	DefaultMaxHistory = 100
)

func main() {
	// 配置命令行参数
	port := flag.Int("port", DefaultPort, "HTTP服务器端口")
	endpoint := flag.String("endpoint", "", "OpenAI兼容API端点 (可选，也可使用OPENAI_ENDPOINT环境变量)")
	apiKey := flag.String("apikey", "", "API密钥 (可选，也可使用OPENAI_API_KEY环境变量)")
	model := flag.String("model", DefaultModel, "要使用的模型名称")
	interval := flag.Int("interval", DefaultInterval, "检测间隔（分钟），0表示不自动检测")
	maxHistory := flag.Int("max-history", DefaultMaxHistory, "保存的历史记录最大数量")

	// 解析命令行参数
	flag.Parse()

	// 获取和验证API端点
	apiEndpoint := getEndpoint(*endpoint)
	if apiEndpoint == "" {
		log.Fatal("必须提供API端点，可通过 --endpoint 参数或设置 OPENAI_ENDPOINT 环境变量")
	}

	// 获取和验证API密钥
	apiKeyValue := getAPIKey(*apiKey)
	if apiKeyValue == "" {
		log.Fatal("必须提供API密钥，可通过 --apikey 参数或设置 OPENAI_API_KEY 环境变量")
	}

	// 创建检测器配置
	config := detector.Config{
		Endpoint:    apiEndpoint,
		APIKey:      apiKeyValue,
		Model:       *model,
		Interval:    *interval,
		MaxHistory:  *maxHistory,
		SaveRawResp: true,
	}

	// 创建并启动服务器
	startServer(config, *port)
}

// getEndpoint 获取API端点，优先使用命令行参数，然后是环境变量
func getEndpoint(cmdEndpoint string) string {
	if cmdEndpoint != "" {
		return cmdEndpoint
	}
	return os.Getenv("OPENAI_ENDPOINT")
}

// getAPIKey 获取API密钥，优先使用命令行参数，然后是环境变量
func getAPIKey(cmdKey string) string {
	if cmdKey != "" {
		return cmdKey
	}
	return os.Getenv("OPENAI_API_KEY")
}

// startServer 创建并启动API服务器
func startServer(config detector.Config, port int) {
	// 创建服务器实例
	server := api.NewServer(config)

	// 构建监听地址
	addr := fmt.Sprintf(":%d", port)

	// 打印启动信息
	log.Printf("API真实性检测服务已启动")
	log.Printf("监听端口: %d", port)
	log.Printf("Web界面: http://localhost:%d", port)

	// 启动服务器
	if err := server.Run(addr); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
