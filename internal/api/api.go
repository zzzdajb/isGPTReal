package api

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"github.com/user/isGPTReal/internal/detector"
)

// Server 表示API服务器
type Server struct {
	router              *gin.Engine        // HTTP路由器
	detector            *detector.Detector // API检测器
	cron                *cron.Cron         // 定时任务管理器
	config              detector.Config    // 配置信息
	cronID              cron.EntryID       // 当前运行的定时任务ID
	mu                  sync.Mutex         // 保护并发访问定时任务
	detectingInProgress bool               // 检测是否正在进行中
	detectionStartTime  time.Time          // 检测开始时间
}

// NewServer 创建一个新的API服务器
func NewServer(config detector.Config) *Server {
	// 设置Gin模式为Release模式，减少不必要的日志输出
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()
	detect := detector.NewDetector(config)

	server := &Server{
		router:              router,
		detector:            detect,
		cron:                cron.New(),
		config:              config,
		detectingInProgress: false,
	}

	// 设置API路由
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

	// API路由组
	api := s.router.Group("/api")
	{
		// 配置相关API
		api.GET("/config", s.getConfig)
		api.POST("/config", s.updateConfig)

		// 检测结果相关API
		api.GET("/results", s.getResults)
		api.GET("/results/latest", s.getLatestResult)

		// 检测控制API
		api.POST("/detect", s.detectNow)

		// 定时任务控制API
		api.POST("/schedule/start", s.startSchedule)
		api.POST("/schedule/stop", s.stopSchedule)
	}
}

// Run 启动API服务器
func (s *Server) Run(addr string) error {
	// 启动定时任务管理器
	s.cron.Start()

	// 如果配置了定时检测间隔，则启动定时任务
	if s.config.Interval > 0 {
		s.startScheduleWithInterval(s.config.Interval)
	}

	// 启动HTTP服务器
	return s.router.Run(addr)
}

// getConfig 返回当前配置
func (s *Server) getConfig(c *gin.Context) {
	// 直接返回配置信息
	c.JSON(http.StatusOK, s.config)
}

// updateConfig 更新配置
func (s *Server) updateConfig(c *gin.Context) {
	var newConfig detector.Config
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置参数: " + err.Error()})
		return
	}

	// 保存原有的定时检测间隔
	oldInterval := s.config.Interval

	// 更新检测器配置
	s.detector.UpdateConfig(newConfig)

	// 更新本地配置副本
	s.config = newConfig

	// 如果定时检测间隔有变化，调整定时任务
	if oldInterval != newConfig.Interval {
		// 停止现有定时任务
		s.stopScheduleInternal()

		// 如果新的间隔大于0，启动新的定时任务
		if newConfig.Interval > 0 {
			s.startScheduleWithInterval(newConfig.Interval)
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已更新"})
}

// getResults 返回所有检测结果
func (s *Server) getResults(c *gin.Context) {
	c.JSON(http.StatusOK, s.detector.GetResults())
}

// getLatestResult 返回最新的检测结果
func (s *Server) getLatestResult(c *gin.Context) {
	// 检查是否有检测正在进行中
	s.mu.Lock()
	isDetecting := s.detectingInProgress
	startTime := s.detectionStartTime
	s.mu.Unlock()

	// 获取最新结果
	result := s.detector.GetLatestResult()

	// 如果检测正在进行中，且结果是在检测开始前的，返回检测中状态
	if isDetecting && (result == nil || result.Timestamp.Before(startTime)) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "detecting",
			"message":   "检测正在进行中",
			"timestamp": time.Now(),
		})
		return
	}

	// 没有结果
	if result == nil {
		c.JSON(http.StatusOK, gin.H{"message": "尚无检测结果"})
		return
	}

	// 返回检测结果
	c.JSON(http.StatusOK, result)
}

// detectNow 执行一次立即检测
func (s *Server) detectNow(c *gin.Context) {
	// 设置检测状态为进行中
	s.mu.Lock()
	s.detectingInProgress = true
	s.detectionStartTime = time.Now()
	s.mu.Unlock()

	// 异步执行检测，避免阻塞HTTP请求
	go func() {
		s.detector.DetectOnce()

		// 检测完成，更新状态
		s.mu.Lock()
		s.detectingInProgress = false
		s.mu.Unlock()
	}()

	c.JSON(http.StatusOK, gin.H{"message": "检测已启动"})
}

// startSchedule 启动定时检测
func (s *Server) startSchedule(c *gin.Context) {
	// 从请求参数中获取间隔时间
	type ScheduleRequest struct {
		Interval int `json:"interval" binding:"required,min=1"`
	}

	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的间隔参数: " + err.Error()})
		return
	}

	// 加锁保护并发访问
	s.mu.Lock()
	defer s.mu.Unlock()

	// 停止现有的定时任务
	if s.cronID != 0 {
		s.cron.Remove(s.cronID)
		s.cronID = 0
	}

	// 更新配置
	s.config.Interval = req.Interval
	s.detector.UpdateConfig(s.config)

	// 启动新的定时任务
	schedule := fmt.Sprintf("@every %dm", req.Interval)
	id, err := s.cron.AddFunc(schedule, func() {
		s.detector.DetectOnce()
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启动定时任务失败: " + err.Error()})
		return
	}

	s.cronID = id
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("定时检测已启动，间隔为%d分钟", req.Interval)})
}

// startScheduleWithInterval 使用指定间隔启动定时检测
func (s *Server) startScheduleWithInterval(minutes int) {
	// 加锁保护并发访问
	s.mu.Lock()
	defer s.mu.Unlock()

	// 停止现有的定时任务
	if s.cronID != 0 {
		s.cron.Remove(s.cronID)
		s.cronID = 0
	}

	// 启动新的定时任务
	schedule := fmt.Sprintf("@every %dm", minutes)
	id, err := s.cron.AddFunc(schedule, func() {
		log.Printf("执行定时检测任务，间隔%d分钟", minutes)

		// 设置检测状态为进行中
		s.mu.Lock()
		s.detectingInProgress = true
		s.detectionStartTime = time.Now()
		s.mu.Unlock()

		// 执行检测并等待结果
		result := s.detector.DetectOnce()

		// 检测完成，更新状态
		s.mu.Lock()
		s.detectingInProgress = false
		s.mu.Unlock()

		// 记录检测结果
		if result.IsRealAPI {
			log.Printf("定时检测完成: 真实API")
		} else {
			log.Printf("定时检测完成: 中转API")
		}
	})

	if err != nil {
		log.Printf("启动定时任务失败: %v", err)
		return
	}

	s.cronID = id
	log.Printf("定时检测已启动，间隔为%d分钟", minutes)
}

// stopSchedule 停止定时检测
func (s *Server) stopSchedule(c *gin.Context) {
	s.stopScheduleInternal()

	// 更新配置
	s.config.Interval = 0
	s.detector.UpdateConfig(s.config)

	c.JSON(http.StatusOK, gin.H{"message": "定时检测已停止"})
}

// stopScheduleInternal 内部使用的停止定时任务的函数
func (s *Server) stopScheduleInternal() {
	// 加锁保护并发访问
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果有运行中的定时任务，停止它
	if s.cronID != 0 {
		s.cron.Remove(s.cronID)
		s.cronID = 0
	}
}
