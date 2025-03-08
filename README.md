# isGPTReal

[![Go Version](https://img.shields.io/badge/Go-1.16%2B-blue)](https://golang.org/)
[![License](https://img.shields.io/badge/License-GPL--3.0-green)](LICENSE)

[English](README_EN.md) | 简体中文

## 📝 项目简介

isGPTReal 是一个强大的工具，用于验证 OpenAI 兼容 API 的真实性。在中国大陆由于无法直接访问 OpenAI 服务，许多用户会使用"中转 API"。这些服务可能使用逆向工程的方式实现，可能存在功能缺失或性能问题。本工具帮助你验证所使用的 API 是否为真实的官方服务。

### ✨ 主要特性

- 🔍 全面的 API 真实性检测
  - Token 限制检测
  - Logprobs 支持检测
  - 多结果返回检测
  - 停止序列功能检测
- 🌐 美观的 Web 界面
- ⏱️ 支持单次检测和定时自动检测
- 📊 检测历史记录查看
- 🔬 API 原始响应内容查看
- 🚀 一键部署，使用简单

## 🛠️ 技术实现

### 检测原理

本工具通过测试以下特性来判断 API 真实性：

| 特性 | 检测内容 | 重要性 |
|------|---------|--------|
| max_tokens | Token 数量限制处理 | ⭐⭐⭐ |
| logprobs | logprobs 信息支持 | ⭐⭐⭐⭐ |
| n 参数 | 多结果返回能力 | ⭐⭐ |
| stop 参数 | 停止序列功能实现 | ⭐⭐⭐ |

## 🚀 快速开始

### 环境要求

- Go 1.16+
- 有效的 OpenAI API 密钥
- Windows/Linux/macOS 系统

### 部署方式

#### 1️⃣ 使用预编译文件（推荐）

```bash
# Windows
isGPTReal.exe --endpoint="https://api.openai.com/v1/chat/completions" --apikey="你的API密钥" --model="gpt-4o-mini"

# Linux/macOS
./isGPTReal --endpoint="https://api.openai.com/v1/chat/completions" --apikey="你的API密钥" --model="gpt-4o-mini"
```

#### 2️⃣ 从源码编译

```bash
git clone https://github.com/zzzdajb/isGPTReal.git
cd isGPTReal
go build -o isGPTReal ./cmd
```

#### 3️⃣ 使用便捷脚本

Windows：
```batch
# 编辑 run.bat 设置环境变量
set OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
set OPENAI_API_KEY=你的API密钥
```

Linux/macOS：
```bash
# 编辑 run.sh 设置环境变量
export OPENAI_ENDPOINT=https://api.openai.com/v1/chat/completions
export OPENAI_API_KEY=你的API密钥
```

### ⚙️ 配置参数

| 参数 | 说明 | 默认值 |
|------|------|--------|
| --endpoint | API 端点 URL | - |
| --apikey | API 密钥 | - |
| --model | 使用的模型 | gpt-4o-mini |
| --interval | 自动检测间隔(分钟) | 0 |
| --port | Web 服务端口 | 8080 |

## 📖 使用指南

1. 启动程序后访问 `http://localhost:8080`
2. 在配置页面填写必要信息：
   - API 端点
   - API 密钥
   - 模型名称
   - 检测间隔（可选）
3. 选择检测模式：
   - 点击"立即检测"进行单次检测
   - 设置间隔并启动定时检测

## ⚠️ 注意事项

- 检测结果仅供参考，不同 API 实现可能影响准确性
- 请确保使用的模型名称与 API 提供商支持的一致
- 程序使用内存存储，重启后数据会丢失
- 建议定期进行检测，及时发现 API 变化

## 📄 许可证

本项目采用 [GPL-3.0](LICENSE) 许可证。

## 🙏 鸣谢

家贫，无以致Cursor以得，每假借于试用之家（狗头），因此特别感谢 [Cursor](https://cursor.sh/) 对本项目的支持！

---

如有问题或建议，欢迎提交 Issue 或 Pull Request。