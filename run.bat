@echo off
REM 设置UTF-8编码
chcp 65001 > nul

echo 正在启动 OpenAI API 真实性检测工具...

REM 如果需要，可以在这里设置环境变量
REM set OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
REM set OPENAI_API_KEY=your-api-key-here

set OPENAI_ENDPOINT=
set OPENAI_API_KEY=

echo 正在编译程序...
go build -o isGPTReal.exe ./cmd

REM 启动程序
echo 启动完成，请在浏览器中访问: http://localhost:8080
isGPTReal.exe --port=8080
