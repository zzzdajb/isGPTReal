#!/bin/bash

echo "正在启动 OpenAI API 真实性检测工具..."

# 如果需要，可以在这里设置环境变量
# export OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
# export OPENAI_API_KEY=your-api-key-here

export OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
export OPENAI_API_KEY=sk-proj-1234567890

echo "正在编译程序..."
go build -o isGPTReal ./cmd

# 添加执行权限
chmod +x ./isGPTReal

# 启动程序
echo "启动完成，请在浏览器中访问: http://localhost:8080"
./isGPTReal --port=8080 