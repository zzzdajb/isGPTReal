package detector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tiktoken-go/tokenizer"
)

// Result 表示一次检测的结果
type Result struct {
	Timestamp    time.Time `json:"timestamp"`
	Endpoint     string    `json:"endpoint"`
	IsRealAPI    bool      `json:"is_real_api"`
	MaxTokensOK  bool      `json:"max_tokens_ok"`
	LogprobsOK   bool      `json:"logprobs_ok"`
	MultipleOK   bool      `json:"multiple_ok"`
	StopSequence bool      `json:"stop_sequence_ok"`
	Error        string    `json:"error,omitempty"`
	RawResponse  string    `json:"raw_response,omitempty"`

	// Token相关信息
	LocalTokenCount int `json:"local_token_count,omitempty"` // 本地计算的token数量
	APITokenCount   int `json:"api_token_count,omitempty"`   // API返回的token数量
	APITotalTokens  int `json:"api_total_tokens,omitempty"`  // API返回的总token数量（包括输入和输出）
}

// Config 表示检测器的配置
type Config struct {
	Endpoint    string `json:"endpoint"`          // OpenAI兼容API的端点URL
	APIKey      string `json:"api_key"`           // API访问密钥
	Model       string `json:"model"`             // 使用的模型名称
	Interval    int    `json:"interval"`          // 自动检测间隔（分钟），0表示不自动检测
	MaxHistory  int    `json:"max_history"`       // 保存的最大历史记录数
	SaveRawResp bool   `json:"save_raw_response"` // 是否保存原始响应
}

// Detector 表示API检测器
type Detector struct {
	config     Config
	results    []Result
	mu         sync.RWMutex // 保护并发访问results
	httpClient *http.Client
}

// NewDetector 创建一个新的检测器实例
func NewDetector(config Config) *Detector {
	// 设置默认的历史记录数
	if config.MaxHistory <= 0 {
		config.MaxHistory = 100
	}

	return &Detector{
		config:     config,
		results:    make([]Result, 0, config.MaxHistory), // 预分配容量
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetResults 返回检测结果历史
func (d *Detector) GetResults() []Result {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.results
}

// GetLatestResult 返回最新的检测结果
func (d *Detector) GetLatestResult() *Result {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if len(d.results) == 0 {
		return nil
	}

	result := d.results[len(d.results)-1]
	return &result
}

// DetectOnce 执行一次完整的API检测
func (d *Detector) DetectOnce() Result {
	result := Result{
		Timestamp: time.Now(),
		Endpoint:  d.config.Endpoint,
		IsRealAPI: false,
	}

	// 按顺序运行所有检测
	maxTokensOK, localTokens, apiTokens, totalTokens, maxTokensErr := d.checkMaxTokens()
	logprobsOK, logprobsErr := d.checkLogprobs()
	multipleOK, multipleErr := d.checkMultipleResponses()
	stopOK, stopErr := d.checkStopSequence()

	// 设置检测结果
	result.MaxTokensOK = maxTokensOK
	result.LogprobsOK = logprobsOK
	result.MultipleOK = multipleOK
	result.StopSequence = stopOK

	// 设置Token信息
	result.LocalTokenCount = localTokens
	result.APITokenCount = apiTokens
	result.APITotalTokens = totalTokens

	// 如果所有检测都通过，则认为是真实API
	result.IsRealAPI = maxTokensOK && logprobsOK && multipleOK && stopOK

	// 收集错误信息
	errorMsgs := []string{}
	if maxTokensErr != nil {
		errorMsgs = append(errorMsgs, fmt.Sprintf("Max tokens测试错误: %v", maxTokensErr))
	}
	if logprobsErr != nil {
		errorMsgs = append(errorMsgs, fmt.Sprintf("Logprobs测试错误: %v", logprobsErr))
	}
	if multipleErr != nil {
		errorMsgs = append(errorMsgs, fmt.Sprintf("Multiple测试错误: %v", multipleErr))
	}
	if stopErr != nil {
		errorMsgs = append(errorMsgs, fmt.Sprintf("Stop sequence测试错误: %v", stopErr))
	}

	// 合并错误信息
	if len(errorMsgs) > 0 {
		result.Error = strings.Join(errorMsgs, "; ")
	}

	// 保存结果
	d.saveResult(result)

	return result
}

// saveResult 保存检测结果到历史记录
func (d *Detector) saveResult(result Result) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// 添加结果
	d.results = append(d.results, result)

	// 如果超过最大历史记录数，删除最早的记录
	if len(d.results) > d.config.MaxHistory {
		d.results = d.results[len(d.results)-d.config.MaxHistory:]
	}
}

// checkMaxTokens 检查max_tokens参数是否生效
func (d *Detector) checkMaxTokens() (bool, int, int, int, error) {
	// 构造请求，限制最大token数为10
	content := "# 人工智能的历史、现状与未来发展趋势\n\n人工智能（Artificial Intelligence，简称AI）"
	maxTokens := 10

	req := map[string]interface{}{
		"model":      d.config.Model,
		"messages":   []map[string]string{{"role": "user", "content": content}},
		"max_tokens": maxTokens,
	}

	// 计算本地token数
	localTokenCount, err := countTokens(content)
	if err != nil {
		log.Printf("计算本地token失败: %v", err)
		localTokenCount = -1
	}

	var response map[string]interface{}
	err = d.makeRequest(req, &response)
	if err != nil {
		return false, localTokenCount, 0, 0, err
	}

	// 获取API返回的token数量
	apiTokenCount := 0
	apiTotalTokens := 0
	var returnedContent string

	if usage, ok := response["usage"].(map[string]interface{}); ok {
		if ct, ok := usage["completion_tokens"].(float64); ok {
			apiTokenCount = int(ct)
		}
		if tt, ok := usage["total_tokens"].(float64); ok {
			apiTotalTokens = int(tt)
		}
	}

	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					returnedContent = content
				}
			}
		}
	}

	// 简洁的日志输出
	log.Printf("MaxTokens检测: 目标限制=%d, 本地计算=%d, API返回=%d, 总tokens=%d",
		maxTokens, localTokenCount, apiTokenCount, apiTotalTokens)
	log.Printf("MaxTokens检测: 返回内容=%q", returnedContent)

	// 检查返回的token数是否符合限制
	// 有些API会精确遵循max_tokens，有些则会返回略少一些token
	return apiTokenCount <= maxTokens, localTokenCount, apiTokenCount, apiTotalTokens, nil
}

// checkLogprobs 检查logprobs参数是否生效
func (d *Detector) checkLogprobs() (bool, error) {
	// 构造请求，要求返回logprobs
	req := map[string]interface{}{
		"model":        d.config.Model,
		"messages":     []map[string]string{{"role": "user", "content": "What is the capital of France?"}},
		"logprobs":     true,
		"top_logprobs": 5,
	}

	var response map[string]interface{}
	err := d.makeRequest(req, &response)
	if err != nil {
		return false, err
	}

	// 检查响应中是否包含logprobs
	hasLogprobs := false
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			_, hasLogprobs = choice["logprobs"].(map[string]interface{})

			// 简洁的日志输出
			log.Printf("Logprobs检测: 请求logprobs=true, 响应包含logprobs=%v", hasLogprobs)

			return hasLogprobs, nil
		}
	}

	return false, fmt.Errorf("无法解析响应格式")
}

// checkMultipleResponses 检查n参数是否生效
func (d *Detector) checkMultipleResponses() (bool, error) {
	// 构造请求多个回答的请求
	req := map[string]interface{}{
		"model":       d.config.Model,
		"messages":    []map[string]string{{"role": "user", "content": "Tell me a short joke"}},
		"n":           3,   // 请求3个不同的回答
		"temperature": 1.0, // 高温度增加多样性
	}

	var response map[string]interface{}
	err := d.makeRequest(req, &response)
	if err != nil {
		return false, err
	}

	// 检查是否返回了多个回答
	n := 0
	if choices, ok := response["choices"].([]interface{}); ok {
		n = len(choices)

		// 简洁的日志输出
		log.Printf("Multiple检测: 请求n=3, 实际返回n=%d", n)

		return n > 1, nil
	}

	return false, fmt.Errorf("无法解析响应格式")
}

// checkStopSequence 检查stop参数是否生效
func (d *Detector) checkStopSequence() (bool, error) {
	// 定义一个特殊的停止序列
	stopSeq := "THE_END"

	// 构造请求，包含stop参数
	req := map[string]interface{}{
		"model":    d.config.Model,
		"messages": []map[string]string{{"role": "user", "content": "写一个短篇故事，不要包含THE_END这个词。"}},
		"stop":     []string{stopSeq},
	}

	var response map[string]interface{}
	err := d.makeRequest(req, &response)
	if err != nil {
		return false, err
	}

	// 检查返回的文本是否不包含停止序列
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					notContainsStop := !strings.Contains(content, stopSeq)

					// 简洁的日志输出
					log.Printf("Stop检测: 设置stop=%s, 响应不包含stop=%v", stopSeq, notContainsStop)

					// 截取一小部分内容
					contentPreview := content
					if len(contentPreview) > 40 {
						contentPreview = contentPreview[:40] + "..."
					}
					log.Printf("Stop检测: 内容片段: %s", contentPreview)

					return notContainsStop, nil
				}
			}
		}
	}

	return false, fmt.Errorf("无法解析响应格式")
}

// makeRequest 向OpenAI API发送请求
func (d *Detector) makeRequest(reqBody map[string]interface{}, response interface{}) error {
	// 序列化请求体
	reqJSON, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("序列化请求体失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", d.config.Endpoint, bytes.NewBuffer(reqJSON))
	if err != nil {
		return fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", d.config.APIKey))

	// 发送请求
	resp, err := d.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应体失败: %w", err)
	}

	// 检查HTTP状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API返回非200状态码: %d, 响应体: %s", resp.StatusCode, truncateString(string(body), 500))
	}

	// 反序列化响应体
	if err := json.Unmarshal(body, response); err != nil {
		return fmt.Errorf("反序列化响应体失败: %w, 响应体: %s", err, truncateString(string(body), 500))
	}

	return nil
}

// UpdateConfig 更新检测器的配置而不创建新的检测器实例
func (d *Detector) UpdateConfig(config Config) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.config = config
}

// CheckEndpointAvailable 检查API端点是否可访问
func (d *Detector) CheckEndpointAvailable() bool {
	req, err := http.NewRequest("GET", d.config.Endpoint, nil)
	if err != nil {
		return false
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode != 0
}

// countTokens 计算文本的token数量
func countTokens(text string) (int, error) {
	enc, err := tokenizer.Get(tokenizer.Cl100kBase)
	if err != nil {
		return 0, fmt.Errorf("获取tokenizer失败: %w", err)
	}

	tokens, _, err := enc.Encode(text)
	if err != nil {
		return 0, fmt.Errorf("编码文本失败: %w", err)
	}

	return len(tokens), nil
}

// 辅助函数：截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
