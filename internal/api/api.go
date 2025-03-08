package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/user/isGPTReal/internal/detector"
)

// Server 表示API服务器
type Server struct {
	router   *gin.Engine
	detector *detector.Detector
	cron     *cron.Cron
	config   detector.Config
	cronID   cron.EntryID
	mu       sync.Mutex
}

// NewServer 创建一个新的API服务器
func NewServer(config detector.Config) *Server {
	router := gin.Default()
	detect := detector.NewDetector(config)

	server := &Server{
		router:   router,
		detector: detect,
		cron:     cron.New(),
		config:   config,
	}

	server.setupRoutes()

	return server
}

// setupRoutes 设置API路由
func (s *Server) setupRoutes() {
	// 配置静态文件服务
	s.router.Static("/static", "./static")
	s.router.LoadHTMLGlob("templates/*")

	// 首页路由
	s.router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"config": s.config,
		})
	})

	// API路由
	api := s.router.Group("/api")
	{
		// 获取配置
		api.GET("/config", s.getConfig)

		// 更新配置
		api.POST("/config", s.updateConfig)

		// 获取检测结果
		api.GET("/results", s.getResults)

		// 获取最新的检测结果
		api.GET("/results/latest", s.getLatestResult)

		// 手动触发一次检测
		api.POST("/detect", s.detectNow)

		// 定时任务控制
		api.POST("/schedule/start", s.startSchedule)
		api.POST("/schedule/stop", s.stopSchedule)
	}
}

// Run 启动API服务器
func (s *Server) Run(addr string) error {
	// 启动定时任务
	s.cron.Start()

	// 如果配置了定时任务，则启动
	if s.config.Interval > 0 {
		s.startScheduleWithInterval(s.config.Interval)
	}

	// 启动HTTP服务器
	return s.router.Run(addr)
}

// getConfig 返回当前配置
func (s *Server) getConfig(c *gin.Context) {
	c.JSON(http.StatusOK, s.config)
}

// updateConfig 更新配置
func (s *Server) updateConfig(c *gin.Context) {
	var newConfig detector.Config
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	s.mu.Lock()
	// 只更新配置
	s.config = newConfig
	// 更新探测器的配置（而不是重新创建）
	s.detector.UpdateConfig(newConfig)

	// 处理定时任务
	if s.cronID != 0 {
		s.stopSchedule(nil) // 停止现有定时任务
	}
	if newConfig.Interval > 0 {
		s.startScheduleWithInterval(newConfig.Interval)
	}
	s.mu.Unlock()

	c.JSON(http.StatusOK, gin.H{"message": "配置已更新"})
}

// getResults 返回所有检测结果
func (s *Server) getResults(c *gin.Context) {
	results := s.detector.GetResults()
	c.JSON(http.StatusOK, results)
}

// getLatestResult 返回最新的检测结果
func (s *Server) getLatestResult(c *gin.Context) {
	result := s.detector.GetLatestResult()
	if result == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "没有检测结果"})
		return
	}

	c.JSON(http.StatusOK, result)
}

// detectNow 立即执行一次检测（异步版本）
func (s *Server) detectNow(c *gin.Context) {
	// 立即返回一个正在处理的响应
	c.JSON(http.StatusAccepted, gin.H{"message": "正在进行检测，请稍后查看结果"})

	// 异步执行检测
	go func() {
		s.detector.DetectOnce()
	}()
}

// startSchedule 开始定时任务
func (s *Server) startSchedule(c *gin.Context) {
	var req struct {
		Interval int `json:"interval"` // 间隔分钟数
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Interval <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "间隔必须大于0"})
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 停止现有定时任务
	if s.cronID != 0 {
		s.cron.Remove(s.cronID)
		s.cronID = 0
	}

	// 更新配置
	s.config.Interval = req.Interval

	// 启动新的定时任务
	s.startScheduleWithInterval(req.Interval)

	c.JSON(http.StatusOK, gin.H{"message": "定时任务已启动"})
}

// startScheduleWithInterval 以指定的间隔启动定时任务
func (s *Server) startScheduleWithInterval(minutes int) {
	// 创建cronspec
	spec := "@every " + time.Duration(minutes).String() + "m"

	// 添加定时任务
	id, err := s.cron.AddFunc(spec, func() {
		s.detector.DetectOnce()
	})

	if err == nil {
		s.cronID = id
	}
}

// stopSchedule 停止定时任务
func (s *Server) stopSchedule(c *gin.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cronID != 0 {
		s.cron.Remove(s.cronID)
		s.cronID = 0
		s.config.Interval = 0
	}

	if c != nil {
		c.JSON(http.StatusOK, gin.H{"message": "定时任务已停止"})
	}
}
