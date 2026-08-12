package rpa

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	applogger "github.com/xingran-next/xingran-go-backend/pkg/logger"
)

// dockerClientTimeout Docker API 客户端超时
const dockerClientTimeout = 30 * time.Second

// DockerClient Docker 客户端接口
type DockerClient interface {
	// ScaleUp 扩容 - 启动新的 Worker 容器
	ScaleUp(ctx context.Context, count int) ([]string, error)

	// ScaleDown 缩容 - 停止指定的 Worker 容器
	ScaleDown(ctx context.Context, containerIDs []string) error

	// ListContainers 列出所有 RPA Worker 容器
	ListContainers(ctx context.Context) ([]DockerContainer, error)

	// GetContainerStats 获取容器统计信息
	GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error)

	// IsHealthy 检查 Docker 服务是否健康
	IsHealthy(ctx context.Context) bool
}

// dockerClientImpl Docker 客户端实现
type dockerClientImpl struct {
	dockerHost    string
	dockerPort    int
	containerName string
	imageName     string
	networkName   string
	httpClient    *http.Client
}

// DockerConfig Docker 配置
type DockerConfig struct {
	DockerHost    string `mapstructure:"docker_host"`    // Docker 主机地址
	DockerPort    int    `mapstructure:"docker_port"`    // Docker API 端口
	ContainerName string `mapstructure:"container_name"` // 容器名称前缀
	ImageName     string `mapstructure:"image_name"`     // 镜像名称
	NetworkName   string `mapstructure:"network_name"`   // 网络名称
}

// NewDockerClient 创建 Docker 客户端
func NewDockerClient(config *DockerConfig) DockerClient {
	return &dockerClientImpl{
		dockerHost:    config.DockerHost,
		dockerPort:    config.DockerPort,
		containerName: config.ContainerName,
		imageName:     config.ImageName,
		networkName:   config.NetworkName,
		httpClient: &http.Client{
			Timeout: dockerClientTimeout,
		},
	}
}

// getBaseURL 获取 Docker API 基础 URL
// 当 dockerHost 未配置时返回错误，避免静默连接到本地 root Docker socket
// (多租户隔离要求)
func (c *dockerClientImpl) getBaseURL() (string, error) {
	if c.dockerHost == "" {
		return "", fmt.Errorf("Docker 主机地址未配置 (docker_host)")
	}
	return fmt.Sprintf("http://%s:%d/v1.40", c.dockerHost, c.dockerPort), nil
}

// makeDockerRequest 发送 Docker API 请求
func (c *dockerClientImpl) makeDockerRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	baseURL, err := c.getBaseURL()
	if err != nil {
		return nil, err
	}
	url := baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	// Docker API 通常使用 Unix socket 或需要特殊认证
	// 这里使用 TCP 连接，生产环境建议使用 TLS
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求失败: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		errorBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Docker API 错误 (status %d): %s", resp.StatusCode, string(errorBody))
	}

	return resp, nil
}

// ScaleUp 扩容 - 启动新的 Worker 容器
func (c *dockerClientImpl) ScaleUp(ctx context.Context, count int) ([]string, error) {
	var containerIDs []string

	for i := 0; i < count; i++ {
		containerID, err := c.createWorkerContainer(ctx)
		if err != nil {
			// 回滚已创建的容器（尽力而为）
			if len(containerIDs) > 0 {
				_ = c.ScaleDown(ctx, containerIDs)
			}
			return nil, fmt.Errorf("创建容器失败: %w", err)
		}
		containerIDs = append(containerIDs, containerID)
		applogger.Infof("扩容: 创建 Worker 容器 %s", containerID)
	}

	return containerIDs, nil
}

// createWorkerContainer 创建单个 Worker 容器
func (c *dockerClientImpl) createWorkerContainer(ctx context.Context) (string, error) {
	// 生成唯一容器名称
	timestamp := time.Now().Unix()
	containerName := fmt.Sprintf("%s-worker-%d", c.containerName, timestamp)

	// 容器配置
	containerConfig := map[string]interface{}{
		"Image": c.imageName,
		"Env": []string{
			"RPA_WORKER_MODE=docker",
			fmt.Sprintf("RPA_WORKER_NAME=%s", containerName),
		},
		"HostConfig": map[string]interface{}{
			"RestartPolicy": map[string]string{
				"Name": "unless-stopped",
			},
			"NetworkMode": c.networkName,
		},
	}

	configJSON, err := json.Marshal(containerConfig)
	if err != nil {
		return "", fmt.Errorf("序列化容器配置失败: %w", err)
	}

	// 创建容器
	resp, err := c.makeDockerRequest(ctx, "POST", "/containers/create?name="+containerName, strings.NewReader(string(configJSON)))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		ID       string   `json:"Id"`
		Warnings []string `json:"Warnings"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("解析响应失败: %w", err)
	}

	// 启动容器
	startResp, err := c.makeDockerRequest(ctx, "POST", "/containers/"+result.ID+"/start", nil)
	if err != nil {
		// 清理已创建但未启动的容器
		if _, cleanupErr := c.makeDockerRequest(ctx, "DELETE", "/containers/"+result.ID, nil); cleanupErr != nil {
			applogger.Warnf("清理未启动的容器 %s 失败: %v", result.ID, cleanupErr)
		}
		return "", fmt.Errorf("启动容器失败: %w", err)
	}
	startResp.Body.Close()

	return result.ID, nil
}

// ScaleDown 缩容 - 停止指定的 Worker 容器
func (c *dockerClientImpl) ScaleDown(ctx context.Context, containerIDs []string) error {
	var errs []string

	for _, containerID := range containerIDs {
		// 先停止容器（优雅关闭，超时 10 秒）
		stopTimeout := 10
		stopPath := fmt.Sprintf("/containers/%s/stop?t=%d", containerID, stopTimeout)
		resp, err := c.makeDockerRequest(ctx, "POST", stopPath, nil)
		if err != nil {
			applogger.Warnf("停止容器 %s 失败: %v", containerID, err)
			errs = append(errs, fmt.Sprintf("%s: %v", containerID, err))
			continue
		}
		resp.Body.Close()

		// 删除容器
		delResp, err := c.makeDockerRequest(ctx, "DELETE", "/containers/"+containerID, nil)
		if err != nil {
			applogger.Warnf("删除容器 %s 失败: %v", containerID, err)
			errs = append(errs, fmt.Sprintf("%s: %v", containerID, err))
			continue
		}
		delResp.Body.Close()

		applogger.Infof("缩容: 停止并删除 Worker 容器 %s", containerID)
	}

	if len(errs) > 0 {
		return fmt.Errorf("部分容器操作失败: %s", strings.Join(errs, "; "))
	}

	return nil
}

// DockerContainer Docker 容器信息
type DockerContainer struct {
	ID        string            `json:"Id"`
	Names     []string          `json:"Names"`
	Image     string            `json:"Image"`
	ImageName string            `json:"ImageName"`
	State     string            `json:"State"`
	Status    string            `json:"Status"`
	Created   int64             `json:"Created"`
	Ports     []Port            `json:"Ports"`
	Labels    map[string]string `json:"Labels"`
}

// Port 端口信息
type Port struct {
	IP          string `json:"IP"`
	PrivatePort int    `json:"PrivatePort"`
	PublicPort  int    `json:"PublicPort"`
	Type        string `json:"Type"`
}

// ListContainers 列出所有 RPA Worker 容器
func (c *dockerClientImpl) ListContainers(ctx context.Context) ([]DockerContainer, error) {
	resp, err := c.makeDockerRequest(ctx, "GET", "/containers/json?all=true", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var containers []DockerContainer
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 过滤出 RPA Worker 容器
	var rpaContainers []DockerContainer
	for _, container := range containers {
		for _, name := range container.Names {
			if strings.Contains(name, c.containerName) {
				rpaContainers = append(rpaContainers, container)
				break
			}
		}
	}

	return rpaContainers, nil
}

// ContainerStats 容器统计信息
type ContainerStats struct {
	ContainerID string
	CPUPercent  float64
	MemoryUsage int64
	MemoryLimit int64
	NetworkRx   int64
	NetworkTx   int64
}

// GetContainerStats 获取容器统计信息
func (c *dockerClientImpl) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	resp, err := c.makeDockerRequest(ctx, "GET", "/containers/"+containerID+"/stats?stream=false", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var stats struct {
		CPUStats struct {
			CPUUsage struct {
				TotalUsage  uint64   `json:"total_usage"`
				PercpuUsage []uint64 `json:"percpu_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_usage"`
			OnlineCPUs  uint64 `json:"online_cpus"`
		} `json:"cpu_stats"`
		PreCPUStats struct {
			CPUUsage struct {
				TotalUsage uint64 `json:"total_usage"`
			} `json:"cpu_usage"`
			SystemUsage uint64 `json:"system_usage"`
			OnlineCPUs  uint64 `json:"online_cpus"`
		} `json:"precpu_stats"`
		MemoryStats struct {
			Usage    uint64 `json:"usage"`
			MaxUsage uint64 `json:"max_usage"`
			Limit    uint64 `json:"limit"`
		} `json:"memory_stats"`
		Networks map[string]struct {
			RxBytes uint64 `json:"rx_bytes"`
			TxBytes uint64 `json:"tx_bytes"`
		} `json:"networks"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("解析统计信息失败: %w", err)
	}

	// 计算 CPU 使用率
	cpuPercent := 0.0
	cpuDelta := stats.CPUStats.CPUUsage.TotalUsage - stats.PreCPUStats.CPUUsage.TotalUsage
	systemDelta := stats.CPUStats.SystemUsage - stats.PreCPUStats.SystemUsage

	if systemDelta > 0 {
		cpuPercent = float64(cpuDelta) / float64(systemDelta) * 100.0
	}

	// 网络统计
	var networkRx, networkTx int64
	for _, netStats := range stats.Networks {
		networkRx += int64(netStats.RxBytes)
		networkTx += int64(netStats.TxBytes)
	}

	return &ContainerStats{
		ContainerID: containerID,
		CPUPercent:  cpuPercent,
		MemoryUsage: int64(stats.MemoryStats.Usage),
		MemoryLimit: int64(stats.MemoryStats.Limit),
		NetworkRx:   networkRx,
		NetworkTx:   networkTx,
	}, nil
}

// IsHealthy 检查 Docker 服务是否健康
func (c *dockerClientImpl) IsHealthy(ctx context.Context) bool {
	resp, err := c.makeDockerRequest(ctx, "GET", "/_ping", nil)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// NewMockDockerClient 创建模拟 Docker 客户端（用于测试）
func NewMockDockerClient() DockerClient {
	return &mockDockerClient{
		containers: make(map[string]*DockerContainer),
	}
}

// mockDockerClient 模拟 Docker 客户端
type mockDockerClient struct {
	containers map[string]*DockerContainer
	counter    int
}

func (m *mockDockerClient) ScaleUp(ctx context.Context, count int) ([]string, error) {
	var ids []string
	for i := 0; i < count; i++ {
		m.counter++
		id := fmt.Sprintf("mock-container-%d", m.counter)
		m.containers[id] = &DockerContainer{
			ID:    id,
			Names: []string{id},
			State: "running",
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (m *mockDockerClient) ScaleDown(ctx context.Context, containerIDs []string) error {
	for _, id := range containerIDs {
		delete(m.containers, id)
	}
	return nil
}

func (m *mockDockerClient) ListContainers(ctx context.Context) ([]DockerContainer, error) {
	var result []DockerContainer
	for _, c := range m.containers {
		result = append(result, *c)
	}
	return result, nil
}

func (m *mockDockerClient) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	return &ContainerStats{
		ContainerID: containerID,
		CPUPercent:  10.5,
		MemoryUsage: 100 * 1024 * 1024,
		MemoryLimit: 1024 * 1024 * 1024,
	}, nil
}

func (m *mockDockerClient) IsHealthy(ctx context.Context) bool {
	// 检查环境变量，如果设置了 MOCK_DOCKER_UNHEALTHY 则返回 false
	return os.Getenv("MOCK_DOCKER_UNHEALTHY") == ""
}
