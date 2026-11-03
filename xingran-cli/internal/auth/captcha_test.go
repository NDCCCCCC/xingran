package auth

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func timeNow() time.Time { return time.Now() }

// makeTestPNG 生成 w x h 的纯色 PNG 并返回 dataURL。
func makeTestPNG(w, h int, c color.Color) string {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}
	return "data:image/png;base64," + pngBase64(buf.Bytes())
}

func TestSaveImage(t *testing.T) {
	dataURL := makeTestPNG(20, 10, color.Black)
	path := filepath.Join(t.TempDir(), "captcha.png")
	if err := SaveImage(dataURL, path); err != nil {
		t.Fatalf("SaveImage 失败: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || len(data) == 0 {
		t.Fatalf("保存的文件不可读: %v", err)
	}
	// PNG 魔数
	if string(data[:4]) != "\x89PNG" {
		t.Fatal("保存的文件不是 PNG")
	}
}

func TestSaveImageEmpty(t *testing.T) {
	if err := SaveImage("", filepath.Join(t.TempDir(), "x.png")); err == nil {
		t.Fatal("空数据应报错")
	}
}

func TestRenderTerminal(t *testing.T) {
	// 黑底白字对比图：上半黑下半白，渲染后应出现暗字符与亮字符
	img := image.NewRGBA(image.Rect(0, 0, 16, 16))
	black := color.RGBA{0, 0, 0, 255}
	white := color.RGBA{255, 255, 255, 255}
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			if y < 8 {
				img.Set(x, y, black)
			} else {
				img.Set(x, y, white)
			}
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	dataURL := "data:image/png;base64," + pngBase64(buf.Bytes())

	ascii, err := RenderTerminal(dataURL, 8)
	if err != nil {
		t.Fatalf("RenderTerminal 失败: %v", err)
	}
	lines := strings.Split(strings.TrimRight(ascii, "\n"), "\n")
	if len(lines) < 2 {
		t.Fatalf("渲染行数不足: %q", ascii)
	}
	if !strings.Contains(ascii, string(asciiRamp[0])) || !strings.Contains(ascii, string(asciiRamp[len(asciiRamp)-1])) {
		t.Fatalf("渲染结果应同时含暗亮字符: %q", ascii)
	}
}

func TestRenderTerminalInvalidDataURL(t *testing.T) {
	if _, err := RenderTerminal("data:image/png;base64,!!!not-base64!!!", 8); err == nil {
		t.Fatal("非法 base64 应报错")
	}
	if _, err := RenderTerminal("", 8); err == nil {
		t.Fatal("空数据应报错")
	}
}
