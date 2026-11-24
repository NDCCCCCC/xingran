// capacitybench Phase 157 / FLEET-03 容量实测工具（宿主机口径）。
//
// 实测口径：本机（Windows，无 Docker）直接以「1 Chromium + N page」为承载单元，
// 对应生产「1 VM = 1 容器 = 1 Chromium = N page」模型中的浏览器层。
// 采集：chrome.exe 进程树工作集（tasklist，零新增依赖）、page 加载失败数
// （崩溃率代理）、进程存活数。
//
// 注意：宿主机数字 ≠ 容器内数字（无 cgroup 隔离、含系统页缓存差异），仅用于校验
// 「1 Chrome + N page」的 RSS 线性关系与崩溃率；容器/VM 内矩阵登记 UAT。
//
// 运行：cd rpa-worker && go run ./cmd/capacitybench -n 1,2,4,8 -rounds 3
package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type sample struct {
	totalWSKB int            // chrome.exe 工作集合计（KB）
	procs     int
	byPID     map[string]int // pid -> 工作集(KB)
}

// sampleChrome 采样全部 chrome.exe 进程（按 PID 区分，由调用方过滤基线）。
func sampleChrome() (sample, error) {
	out, err := exec.Command("tasklist", "/FI", "IMAGENAME eq chrome.exe", "/FO", "CSV", "/NH").Output()
	if err != nil {
		return sample{}, err
	}
	s := sample{byPID: map[string]int{}}
	r := csv.NewReader(strings.NewReader(string(out)))
	recs, err := r.ReadAll()
	if err != nil {
		return sample{}, err
	}
	for _, rec := range recs {
		if len(rec) < 5 || strings.HasPrefix(rec[0], "INFO") {
			continue
		}
		pid := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(rec[1]), `"`))
		ws := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(rec[4]), `"`), `"`)
		ws = strings.ReplaceAll(strings.TrimSuffix(ws, " K"), ",", "")
		if v, err := strconv.Atoi(strings.TrimSpace(ws)); err == nil {
			s.byPID[pid] = v
			s.totalWSKB += v
			s.procs++
		}
	}
	return s, nil
}

// filterBaseline 排除启动前已存在的进程（用户自己的浏览器），只统计本轮启动的进程树。
func filterBaseline(s sample, baseline map[string]bool) sample {
	out := sample{byPID: map[string]int{}}
	for pid, ws := range s.byPID {
		if !baseline[pid] {
			out.totalWSKB += ws
			out.procs++
		}
	}
	return out
}

func main() {
	ns := flag.String("n", "1,2,4,8", "并发档位，逗号分隔")
	rounds := flag.Int("rounds", 3, "每档轮数")
	bin := flag.String("bin", `C:\Program Files\Google\Chrome\Application\chrome.exe`, "Chrome 可执行文件路径")
	flag.Parse()

	var tierList []int
	for _, f := range strings.Split(*ns, ",") {
		v, err := strconv.Atoi(strings.TrimSpace(f))
		if err == nil && v > 0 {
			tierList = append(tierList, v)
		}
	}

	// 中等负载页面：~800 行内容（data URL，无外网依赖）
	url := "data:text/html," + strings.Repeat(
		"<div>capacity bench row with some text content and a <a href='#'>link</a></div>", 800)

	fmt.Println("| N | rounds | maxRSS(MB) | avgRSS(MB) | chromeProcs(max) | pageLoadFail | browserCrash |")
	fmt.Println("|---|---|---|---|---|---|---|")

	for _, n := range tierList {
		var maxKB, sumKB, samples int64
		maxProcs := 0
		loadFails, crashes := 0, 0

		for r := 0; r < *rounds; r++ {
			// 基线：本轮启动前已存在的 chrome.exe PID（用户自己的浏览器，排除）
			baseSnap, err := sampleChrome()
			if err != nil {
				continue
			}
			baseline := make(map[string]bool, len(baseSnap.byPID))
			for pid := range baseSnap.byPID {
				baseline[pid] = true
			}

			// 每轮独立浏览器（对应「一容器一 Chromium」生命周期）
			u, err := launcher.New().
				Bin(*bin).
				Headless(true).
				Set("disable-gpu").
				Set("disable-dev-shm-usage").
				Set("no-sandbox").
				Launch()
			if err != nil {
				crashes++
				continue
			}

			browser := rod.New().ControlURL(u)
			if err := browser.Connect(); err != nil {
				crashes++
				continue
			}

			var pages []*rod.Page
			for i := 0; i < n; i++ {
				p, err := browser.Page(proto.TargetCreateTarget{URL: url})
				if err != nil {
					loadFails++
					continue
				}
				if err := p.WaitLoad(); err != nil {
					loadFails++
				}
				pages = append(pages, p)
			}

			// 采样 10 次 × 500ms（页面已加载的稳态窗口，仅统计本轮启动的进程树）
			for i := 0; i < 10; i++ {
				if s, err := sampleChrome(); err == nil {
					s = filterBaseline(s, baseline)
					if int64(s.totalWSKB) > maxKB {
						maxKB = int64(s.totalWSKB)
					}
					sumKB += int64(s.totalWSKB)
					samples++
					if s.procs > maxProcs {
						maxProcs = s.procs
					}
				}
				time.Sleep(500 * time.Millisecond)
			}

			for _, p := range pages {
				_ = p.Close()
			}
			browser.Close()
			time.Sleep(1 * time.Second)
		}

		avg := int64(0)
		if samples > 0 {
			avg = sumKB / samples
		}
		fmt.Printf("| %d | %d | %d | %d | %d | %d | %d |\n",
			n, *rounds, maxKB/1024, avg/1024, maxProcs, loadFails, crashes)
	}
}
