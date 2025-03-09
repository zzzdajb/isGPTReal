# isGPTReal - OpenAI API 逆向检测工具

[![Go Version](https://img.shields.io/badge/Go-1.24%2B-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-GPL--3.0-green)](LICENSE)

## 📝 项目简介

由于中国大陆地区对 OpenAI API 的访问受到限制，许多用户会使用“中转API”来解决使用问题。然而，这些中转API良莠不齐、鱼龙混杂，许多商家以次充好，使用逆向的API来冒充OpenAI或者Azure的API，本工具工具旨在帮助用户快速检测这些“中转API”的真实性。同时，为了解决商家“前期使用真API，后期掺水”的问题，本工具也提供了定时检测功能，帮助用户及时发现问题。

### ✨ 主要特性

- 🔍 **全面的 API 真实性检测**
  - Token 限制检测 - 验证 API 是否正确实现了 token 数量限制
  - Logprobs 支持检测 - 检查 API 是否支持返回 logprobs 信息
  - 多结果返回检测 - 测试 API 是否实现了多结果(n)参数功能
  - 停止序列功能检测 - 验证 API 是否正确处理停止序列参数
- 🌐 **美观的 Web 界面** - 直观显示检测结果和历史记录
- ⏱️ **灵活的检测模式** - 支持单次检测和定时自动检测
- 📊 **完整的结果分析** - 保存检测历史记录和详细结果
- 🔬 **原始响应查看** - 可查看 API 返回的原始 JSON 响应
- 🚀 **便捷的部署方式** - 支持多种部署方式，使用简单

## 🛠️ 技术实现

### 检测原理

本工具通过测试以下特性来判断 API 真实性：

| 特性 | 检测内容 | 重要性 | 说明 |
|------|---------|--------|------|
| max_tokens | Token 数量限制处理 | ⭐⭐⭐ | 检测 API 是否正确实现了 token 数量限制功能 |
| logprobs | logprobs 信息支持 | ⭐⭐⭐⭐ | 检测 API 是否支持返回 token 概率信息，这是非官方 API 难以实现的功能 |
| n 参数 | 多结果返回能力 | ⭐⭐ | 检测 API 是否能正确处理多结果返回请求 |
| stop 参数 | 停止序列功能实现 | ⭐⭐⭐ | 检测 API 是否能在指定序列处正确停止生成 |

### 技术栈

- **后端框架**: Go 1.24+，使用 Gin 框架实现 Web 服务
- **前端技术**: HTML/CSS/JavaScript，使用 Bootstrap 5 构建响应式界面
- **第三方库**:
  - gin-gonic/gin - Web 框架
  - robfig/cron - 定时任务调度
  - tiktoken-go/tokenizer - OpenAI tokenizer 实现

## 🚀 快速开始

### 环境要求

- Go 1.24+ (仅从源码构建时需要)
- 有效的 OpenAI API 密钥
- Windows/Linux/macOS 系统

### 部署方式

#### 1️⃣ 使用预编译文件（推荐）

从Github Release下载适合您系统的预编译文件，解压以后直接运行：

```bash
# Windows
isGPTReal.exe --endpoint="https://api.openai.com/v1/chat/completions" --apikey="Your_API_Key" --model="gpt-4o-mini"

# Linux/macOS
./isGPTReal --endpoint="https://api.openai.com/v1/chat/completions" --apikey="Your_API_Key" --model="gpt-4o-mini"
```

#### 2️⃣ 从源码编译

```bash
# 克隆仓库
git clone https://github.com/zzzdajb/isGPTReal.git
cd isGPTReal

# 编译
go build -o isGPTReal ./cmd

# 运行
./isGPTReal --endpoint="https://api.openai.com/v1/chat/completions" --apikey="Your_API_Key"
```

#### 3️⃣ 使用便捷脚本

提供了便捷脚本，可通过环境变量配置参数：

Windows：
```batch
# 编辑 run.bat 设置环境变量
set OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
set OPENAI_API_KEY=Your_API_Key

# 运行脚本
run.bat
```

Linux/macOS：
```bash
# 编辑 run.sh 设置环境变量
export OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
export OPENAI_API_KEY=Your_API_Key

# 添加执行权限
chmod +x run.sh

# 运行脚本
./run.sh
```

#### 4️⃣ 使用Docker部署

本项目提供了Docker部署支持，可以通过Docker容器快速部署和运行。

##### 前提条件
- 安装 [Docker](https://docs.docker.com/get-docker/)
- 安装 [Docker Compose](https://docs.docker.com/compose/install/)（可选，用于使用docker-compose.yml）

##### 单容器部署
```bash
# 构建Docker镜像
docker build -t isgptreal .

# 运行容器
docker run -d -p 8080:8080 \
  -e OPENAI_ENDPOINT="https://api.openai.com/v1/chat/completions" \
  -e OPENAI_API_KEY="Your_API_Key" \
  --name isgptreal isgptreal
```

##### 使用Docker Compose部署
1. 创建环境变量文件 `.env`：
```
OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
OPENAI_API_KEY=Your_API_Key
```

2. 运行Docker Compose：
```bash
docker-compose up -d
```

##### 自定义Docker配置
您可以通过命令行参数自定义容器配置：
```bash
docker run -d -p 8080:8080 \
  -e OPENAI_ENDPOINT="https://api.openai.com/v1/chat/completions" \
  -e OPENAI_API_KEY="Your_API_Key" \
  --name isgptreal isgptreal \
  --model="gpt-4" --interval=60 --max-history=200
```

##### 查看容器日志
```bash
# 单容器部署
docker logs -f isgptreal

# Docker Compose部署
docker-compose logs -f
```

##### 停止和删除容器
```bash
# 单容器部署
docker stop isgptreal
docker rm isgptreal

# Docker Compose部署
docker-compose down
```

### ⚙️ 配置参数

| 参数 | 说明 | 默认值 | 环境变量 |
|------|------|--------|---------|
| --endpoint | API URL | https://api.openai.com/v1/chat/completions | OPENAI_ENDPOINT |
| --apikey | API 密钥 | Your_API_Key | OPENAI_API_KEY |
| --model | 使用的模型 | gpt-4o-mini | - |
| --interval | 自动检测间隔(分钟) | 0 | - |
| --port | Web 服务端口 | 8080 | - |
| --max-history | 保存的历史记录最大数量 | 100 | - |

## 📖 使用指南

1. **启动程序**
   - 启动后访问 `http://localhost:8080`（或您设置的端口）

2. **配置检测参数**
   - 在 Web 界面配置页面填写必要信息：
     - API Endpoint
     - API key
     - 模型名称
     - 检测间隔（可选）

3. **进行检测**
   - 点击"立即检测"按钮进行单次检测
   - 设置间隔时间并启动定时检测

4. **查看结果**
   - 检测完成后可查看详细结果和评分
   - 点击"查看原始响应"可以查看 API 返回的 JSON 数据

5. **历史记录**
   - 查看历史检测记录和趋势变化


## ⚠️ 注意事项

- 检测结果仅供参考，不同 API 实现可能影响准确性
- 请确保使用的模型名称与 API 提供商支持的一致
- 程序使用内存存储，重启后数据会丢失
- 建议定期进行检测，及时发现 API 质量变化

## 📄 许可证

本项目采用 [GPL-3.0](LICENSE) 许可证。

## 🙏 鸣谢

家贫，无以致Cursor以得，每假借于试用之家（狗头），因此特别感谢 [Cursor](https://cursor.sh/) 对本项目的支持！

---

如有问题或建议，欢迎提交 Issue 或 Pull Request。（放心大胆地提，反正我又不会写代码，我只会让Cursor看）