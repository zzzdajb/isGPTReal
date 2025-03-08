# isGPTReal - OpenAI API 真实性检测工具

## 项目简介

这个工具用于检测OpenAI兼容API是否为真实的官方API或是"中转API"。由于OpenAI官方不支持在中国大陆提供服务，许多中国用户使用"中转API"服务，而这些服务可能实际使用的是逆向工程的API，功能和性能可能与官方API存在差异。

### 检测原理

本工具通过测试以下四种OpenAI API特性来判断API真实性：

- **max_tokens参数**：检查是否正确处理token数量限制
- **logprobs参数**：检查是否支持并返回logprobs信息
- **n参数**：检查是否能正确处理多个结果返回
- **stop参数**：检查是否能正确实现停止序列功能

### 功能特性

- 单次检测与定时自动检测
- 美观的Web界面实时展示检测结果
- 检测历史记录查看
- 支持查看API原始响应内容
- 前后端一体化设计，易于部署

## 部署教程

### 环境要求

- Go 1.16或更高版本
- 有效的OpenAI API密钥

### 方法一：从源码安装

1. 克隆仓库

```bash
git clone https://github.com/user/isGPTReal.git
cd isGPTReal
```

2. 编译项目

```bash
go build -o isGPTReal ./cmd
```

3. 运行程序

```bash
# 基本运行方式
./isGPTReal

# 或者指定参数运行（包括模型）
./isGPTReal --endpoint="https://api.openai.com/v1/chat/completions" --apikey="你的API密钥" --model="gpt-4o-mini" --port=8080
```

### 方法二：使用预编译的可执行文件

1. 下载适合你操作系统的可执行文件
2. 赋予执行权限（Linux/Mac）或直接运行（Windows）
3. 使用以下命令运行：

Windows:
```
isGPTReal.exe --endpoint="https://api.openai.com/v1/chat/completions" --apikey="你的API密钥" --model="gpt-4o-mini"
```

Linux/Mac:
```
./isGPTReal --endpoint="https://api.openai.com/v1/chat/completions" --apikey="你的API密钥" --model="gpt-4o-mini"
```

### 方法三：使用提供的脚本运行

项目提供了便捷的运行脚本：

- Windows系统：双击运行`run.bat`
- Linux/Mac系统：运行`./run.sh`

你可以编辑这些脚本，在其中设置环境变量：

```
# Windows (run.bat)
set OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
set OPENAI_API_KEY=你的API密钥

# Linux/Mac (run.sh)
export OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
export OPENAI_API_KEY=你的API密钥
```

### 命令行参数

- `--endpoint`：OpenAI兼容API的端点URL
- `--apikey`：API密钥
- `--model`：使用的模型（默认：gpt-3.5-turbo）
  - 支持的模型：gpt-3.5-turbo、gpt-4等OpenAI支持的聊天模型
  - 如果使用第三方API，请确保指定的模型名称与API提供商支持的模型一致
- `--interval`：自动检测间隔，单位为分钟（0表示不自动检测）
- `--port`：Web服务器端口（默认：8080）

## 使用说明

1. 启动程序后，在浏览器中访问 `http://localhost:8080`
2. 在配置页面填写：
   - API端点
   - API密钥
   - 模型名称（默认为gpt-3.5-turbo，可根据需要修改）
   - 检测间隔（可选）
3. 点击"立即检测"按钮进行单次检测
4. 或设置检测间隔并点击"启动定时检测"进行自动检测
5. 查看检测结果和历史记录

## 注意事项

- 本工具仅用于技术研究和学习目的
- 请遵守OpenAI的服务条款和相关法律法规
- 检测结果仅供参考，不同的API实现可能会影响检测准确性
- 确保使用的模型名称与API提供商支持的模型一致，否则可能导致检测失败

## 许可证

MIT

## 免责声明

本工具仅用于技术研究和学习目的，请遵守OpenAI的服务条款和相关法律法规。 