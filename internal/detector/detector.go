package detector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	// 新增token相关信息
	LocalTokenCount int `json:"local_token_count,omitempty"`
	APITokenCount   int `json:"api_token_count,omitempty"`
	APITotalTokens  int `json:"api_total_tokens,omitempty"`
}

// Config 表示检测器的配置
type Config struct {
	Endpoint    string `json:"endpoint"`
	APIKey      string `json:"api_key"`
	Model       string `json:"model"`
	Interval    int    `json:"interval"` // 以分钟为单位
	MaxHistory  int    `json:"max_history"`
	SaveRawResp bool   `json:"save_raw_response"`
}

// Detector 表示API检测器
type Detector struct {
	config     Config
	results    []Result
	mu         sync.RWMutex
	httpClient *http.Client
}

// NewDetector 创建一个新的检测器
func NewDetector(config Config) *Detector {
	if config.MaxHistory <= 0 {
		config.MaxHistory = 100
	}

	return &Detector{
		config:     config,
		results:    make([]Result, 0),
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

	result.MaxTokensOK = maxTokensOK
	result.LogprobsOK = logprobsOK
	result.MultipleOK = multipleOK
	result.StopSequence = stopOK

	// 保存token相关数据
	result.LocalTokenCount = localTokens
	result.APITokenCount = apiTokens
	result.APITotalTokens = totalTokens

	// 汇总结果
	var errors []string
	if maxTokensErr != nil {
		errors = append(errors, fmt.Sprintf("MaxTokens error: %v", maxTokensErr))
	}
	if logprobsErr != nil {
		errors = append(errors, fmt.Sprintf("Logprobs error: %v", logprobsErr))
	}
	if multipleErr != nil {
		errors = append(errors, fmt.Sprintf("Multiple responses error: %v", multipleErr))
	}
	if stopErr != nil {
		errors = append(errors, fmt.Sprintf("Stop sequence error: %v", stopErr))
	}

	if len(errors) > 0 {
		result.Error = fmt.Sprintf("%v", errors)
	}

	// 如果所有检测都通过，则认为是真实的API
	result.IsRealAPI = maxTokensOK && logprobsOK && multipleOK && stopOK

	// 保存结果
	d.mu.Lock()
	d.results = append(d.results, result)
	if len(d.results) > d.config.MaxHistory {
		d.results = d.results[len(d.results)-d.config.MaxHistory:]
	}
	d.mu.Unlock()

	return result
}

// checkMaxTokens 检查max_tokens参数是否生效
func (d *Detector) checkMaxTokens() (bool, int, int, int, error) {
	// 构造请求极少tokens的请求
	shortReq := map[string]interface{}{
		"model":       d.config.Model,
		"messages":    []map[string]string{{"role": "user", "content": "请写一篇非常长的文章，描述人工智能的历史、现状和未来，至少5000字"}},
		"max_tokens":  10,
		"temperature": 0.3,
	}

	var response map[string]interface{}
	err := d.makeRequest(shortReq, &response)
	if err != nil {
		return false, 0, 0, 0, err
	}

	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					// 使用tiktoken计算实际token数（仅用于记录）
					localTokenCount, _ := countTokens(content)

					// 获取API自己报告的token数
					apiTokenCount := 0
					apiTotalTokens := 0
					isApiTokenValid := false
					if usage, ok := response["usage"].(map[string]interface{}); ok {
						if completionTokens, ok := usage["completion_tokens"].(float64); ok {
							apiTokenCount = int(completionTokens)
							isApiTokenValid = true
						}
						if totalTokens, ok := usage["total_tokens"].(float64); ok {
							apiTotalTokens = int(totalTokens)
						}
					}

					// 使用fmt直接打印到控制台，确保能看到
					fmt.Println("\n========== Token检测详细信息 ==========")
					fmt.Printf("目标限制: 10 tokens\n")
					fmt.Printf("本地计算Token数: %d\n", localTokenCount)
					if isApiTokenValid {
						fmt.Printf("API报告的completion_tokens: %d\n", apiTokenCount)
					} else {
						fmt.Printf("API未报告token数量\n")
					}
					fmt.Printf("实际内容: %q\n", content)

					// 打印API返回的token信息（如果有）
					if apiTotalTokens > 0 {
						fmt.Printf("API报告的total_tokens: %d\n", apiTotalTokens)
					}

					// 判断标准：优先使用API报告的token数，如果API没有报告则使用本地计算
					tokenCount := localTokenCount
					if isApiTokenValid {
						tokenCount = apiTokenCount
					}

					fmt.Printf("用于判断的Token数: %d\n", tokenCount)
					fmt.Printf("通过检测: %v\n", tokenCount <= 10)
					fmt.Println("=====================================\n")

					return tokenCount <= 10, localTokenCount, apiTokenCount, apiTotalTokens, nil
				}
			}
		}
	}

	return false, 0, 0, 0, fmt.Errorf("无法解析响应格式")
}

// checkLogprobs 检查logprobs参数是否生效
func (d *Detector) checkLogprobs() (bool, error) {
	// 构造带有logprobs参数的请求
	req := map[string]interface{}{
		"model":        d.config.Model,
		"messages":     []map[string]string{{"role": "user", "content": "Hello"}},
		"logprobs":     true,
		"top_logprobs": 5,
	}

	var response map[string]interface{}
	err := d.makeRequest(req, &response)
	if err != nil {
		return false, err
	}

	// 检查是否包含logprobs
	hasLogprobs := false
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			_, hasLogprobs = choice["logprobs"].(map[string]interface{})

			// 打印详细信息到控制台
			fmt.Println("\n========== Logprobs参数检测 ==========")
			fmt.Printf("请求的logprobs: true\n")
			fmt.Printf("请求的top_logprobs: 5\n")
			fmt.Printf("响应中是否包含logprobs: %v\n", hasLogprobs)

			// 尝试获取更多详细信息
			if hasLogprobs {
				if logprobs, lpOk := choice["logprobs"].(map[string]interface{}); lpOk {
					if topLogprobs, tlpOk := logprobs["top_logprobs"].([]interface{}); tlpOk {
						fmt.Printf("返回的top_logprobs数量: %d\n", len(topLogprobs))
					}
				}
			}

			fmt.Printf("通过检测: %v\n", hasLogprobs)
			fmt.Println("=====================================\n")

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

	// 检查是否返回了多个选择
	if choices, ok := response["choices"].([]interface{}); ok {
		isMultiple := len(choices) > 1

		// 打印详细信息到控制台
		fmt.Println("\n========== 多选项(n)参数检测 ==========")
		fmt.Printf("请求的选项数: 3\n")
		fmt.Printf("实际返回选项数: %d\n", len(choices))
		fmt.Printf("通过检测: %v\n", isMultiple)
		fmt.Println("=====================================\n")

		return isMultiple, nil
	}

	return false, fmt.Errorf("无法解析响应格式")
}

// checkStopSequence 检查stop参数是否生效
func (d *Detector) checkStopSequence() (bool, error) {
	// 构造带有stop序列的请求
	stopSeq := "THE_END"
	req := map[string]interface{}{
		"model":    d.config.Model,
		"messages": []map[string]string{{"role": "user", "content": "请写一个故事，不要包含THE_END这个词"}},
		"stop":     []string{stopSeq},
	}

	var response map[string]interface{}
	err := d.makeRequest(req, &response)
	if err != nil {
		return false, err
	}

	// 检查响应中是否包含stop序列
	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					// 检查内容是否被stop序列截断
					stopSequenceWorks := !strings.Contains(content, stopSeq)

					// 打印详细信息到控制台
					fmt.Println("\n========== Stop参数检测 ==========")
					fmt.Printf("设置的stop序列: %q\n", stopSeq)
					fmt.Printf("返回内容是否不含stop序列: %v\n", stopSequenceWorks)
					fmt.Printf("内容片段: %s\n", truncateString(content, 100)) // 只显示前100个字符
					fmt.Printf("通过检测: %v\n", stopSequenceWorks)
					fmt.Println("=====================================\n")

					return stopSequenceWorks, nil
				}
			}
		}
	}

	return false, fmt.Errorf("无法解析响应格式")
}

// 辅助函数：截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// makeRequest 发送请求到API
func (d *Detector) makeRequest(reqBody map[string]interface{}, response interface{}) error {
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", d.config.Endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", d.config.APIKey))

	resp, err := d.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if d.config.SaveRawResp {
		// 保存原始响应到最后一个结果中
		d.mu.Lock()
		if len(d.results) > 0 {
			d.results[len(d.results)-1].RawResponse = string(body)
		}
		d.mu.Unlock()
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API返回错误状态码: %d, 响应: %s", resp.StatusCode, string(body))
	}

	return json.Unmarshal(body, response)
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

func countTokens(text string) (int, error) {
	enc, err := tokenizer.Get(tokenizer.Cl100kBase) // 使用GPT-3.5/4使用的编码器
	if err != nil {
		return 0, err
	}

	ids, tokens, err := enc.Encode(text)
	if err != nil {
		return 0, err
	}

	// 打印详细的token信息
	fmt.Println("\n=== Token计算详情 ===")
	fmt.Printf("原始文本: %q\n", text)
	fmt.Printf("Token数量: %d\n", len(ids))
	fmt.Println("Token列表:")
	for i, token := range tokens {
		fmt.Printf("[%d] %q (ID: %d)\n", i, token, ids[i])
	}
	fmt.Println("==================\n")

	return len(ids), nil
}
