package deployments

// Phase 157 / FLEET-02：配置漂移校验。
//
// 断言「一 VM 一容器一 Chromium 多 page」的三个真相源保持一致：
//   1. docker-compose.yml / docker-compose.local.yml 的 MAX_CONCURRENCY 默认值
//   2. rpa-worker/configs/config.yaml 的 worker.max_concurrency
//   3. deployments/README.md 容量模型的锚点行（capacity-n: <N>）
//
// 以及 8GB VM 基准资源参数：shm_size 1gb / memory 6G / cpus '4'。
//
// Phase 154 遗留兼容：若 config.yaml 中仍存在非注释的旧键 `max_pages:`，
// 本测试打印 WARN 但不失败（中间态容忍，见 ROADMAP 归属协议第 1 条）；
// 旧键一旦被删除，本测试同样通过（两种状态都合法）。
//
// 运行：cd rpa-worker && go test ./deployments/
// 若本机无 docker，`docker compose config` 解析检查自动跳过并登记。

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// capacityN 容量模型唯一真相：N = max_concurrency = MAX_CONCURRENCY。
const capacityN = "8"

const (
	expectShm    = "1gb"
	expectMemory = "6G"
	expectCPUs   = "'4'"
)

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.FromSlash(rel))
	if err != nil {
		t.Fatalf("读取 %s 失败: %v", rel, err)
	}
	return string(b)
}

// TestCapacityDrift_ComposeConcurrency compose 与 config 的 N 一致。
func TestCapacityDrift_ComposeConcurrency(t *testing.T) {
	for _, compose := range []string{"docker-compose.yml", "docker-compose.local.yml"} {
		text := readRepoFile(t, compose)
		re := regexp.MustCompile(`MAX_CONCURRENCY=\$\{MAX_CONCURRENCY:-(\d+)\}`)
		m := re.FindStringSubmatch(text)
		if m == nil {
			t.Fatalf("%s: 找不到 MAX_CONCURRENCY 默认值声明", compose)
		}
		if m[1] != capacityN {
			t.Errorf("%s: MAX_CONCURRENCY 默认值 = %s, 期望 %s（容量模型漂移）", compose, m[1], capacityN)
		}
	}

	cfg := readRepoFile(t, "../configs/config.yaml")
	re := regexp.MustCompile(`(?m)^\s*max_concurrency:\s*(\d+)`)
	m := re.FindStringSubmatch(cfg)
	if m == nil {
		t.Fatalf("configs/config.yaml: 找不到 worker.max_concurrency")
	}
	if m[1] != capacityN {
		t.Errorf("configs/config.yaml: max_concurrency = %s, 期望 %s（容量模型漂移）", m[1], capacityN)
	}

	// README 容量模型锚点行
	readme := readRepoFile(t, "README.md")
	reAnchor := regexp.MustCompile(`(?m)^<!-- capacity-n:\s*(\d+)\s*-->`)
	ma := reAnchor.FindStringSubmatch(readme)
	if ma == nil {
		t.Errorf("deployments/README.md: 缺少容量锚点行 `<!-- capacity-n: %s -->`（漂移校验的文档真相源）", capacityN)
	} else if ma[1] != capacityN {
		t.Errorf("deployments/README.md: capacity-n 锚点 = %s, 期望 %s", ma[1], capacityN)
	}
}

// TestCapacityDrift_ComposeResources compose 的 8GB VM 基准资源参数。
func TestCapacityDrift_ComposeResources(t *testing.T) {
	for _, compose := range []string{"docker-compose.yml", "docker-compose.local.yml"} {
		text := readRepoFile(t, compose)

		if !strings.Contains(text, "shm_size: "+expectShm) {
			t.Errorf("%s: 缺少 shm_size: %s", compose, expectShm)
		}
		if !strings.Contains(text, "memory: "+expectMemory) {
			t.Errorf("%s: 缺少 limits.memory: %s", compose, expectMemory)
		}
		if !strings.Contains(text, "cpus: "+expectCPUs) {
			t.Errorf("%s: 缺少 limits.cpus: %s", compose, expectCPUs)
		}
		if !strings.Contains(text, "--disable-dev-shm-usage") {
			t.Errorf("%s: CHROME_FLAGS 缺少 --disable-dev-shm-usage（shm 1gb 的前提）", compose)
		}
	}
}

// TestCapacityDrift_LegacyMaxPages 旧键 max_pages 的中间态容忍（存在→WARN，不存在→通过）。
func TestCapacityDrift_LegacyMaxPages(t *testing.T) {
	cfg := readRepoFile(t, "../configs/config.yaml")
	re := regexp.MustCompile(`(?m)^\s*max_pages:\s*\d+`)
	if re.MatchString(cfg) {
		t.Logf("WARN: configs/config.yaml 仍存在旧键 browser.max_pages（Phase 154 已合并进 "+
			"worker.max_concurrency，运行时告警并迁移）。中间态容忍，不阻塞；建议后续清理。")
	}
	// headless 取值必须存在且为 true（无头基准）
	reHeadless := regexp.MustCompile(`(?m)^\s*headless:\s*(true|false)`)
	mh := reHeadless.FindStringSubmatch(cfg)
	if mh == nil {
		t.Errorf("configs/config.yaml: 找不到 browser.headless")
	} else if mh[1] != "true" {
		t.Errorf("configs/config.yaml: browser.headless = %s, 期望 true", mh[1])
	}
}

// TestCapacityDrift_ComposeParse docker compose config 解析（无 docker 则跳过并登记）。
func TestCapacityDrift_ComposeParse(t *testing.T) {
	docker, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("SKIP(登记): 本机无 docker，跳过 docker compose config 解析校验（UAT 环境执行）")
	}
	for _, compose := range []string{"docker-compose.yml", "docker-compose.local.yml"} {
		cmd := exec.Command(docker, "compose", "-f", compose, "config", "--quiet")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Errorf("%s: docker compose config 解析失败: %v\n%s", compose, err, string(out))
		}
	}
}
