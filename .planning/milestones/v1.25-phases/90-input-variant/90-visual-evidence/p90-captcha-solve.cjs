// Phase 90 —— 滑块验证码缺口自动识别（用于巡检的登录前置）
//
// 原理（读 pkg/captcha/slider.go 得到，非猜测）：
//   `drawShadowOnBackgroundWithShape` 在背景图上于 (xPos, yPos) 处**叠加深色阴影**
//   （「变暗到 25%」）并调 `drawGapBorder` 绘制缺口边框 →
//   缺口区域在背景图上表现为一块**明显偏暗**的矩形（相对同一行其他像素）。
//   拼图块本身是从背景图**同位置**提取的（`generatePieceFromBackground`），
//   且带白色边框 + 高光 → 可作为第二重校验。
//
// 策略（逐列亮度曲线法，零依赖）：
//   1. 取背景图，逐列算「该列与该行带区的平均亮度」；
//   2. 缺口所在列的亮度显著低于整体中位数 → 找出连续暗列区间；
//   3. 取区间左端作为缺口 x 的候选；
//   4. 用滑块实际拖动后的服务端响应做校验（失败则换下一个候选）。
//
// ⚠️ 坐标换算：前端容器宽度 = 显示宽度，后端原始图宽 = 400（`SliderCaptcha.Width`）。
//   前端提交前会自己做 `displayX * (naturalWidth / clientWidth)` 换算，
//   故本脚本只需给出**显示坐标**（与 `dragStateRef.current.currentX` 同尺度）。
//
// 用法：node p90-captcha-solve.cjs <bgBase64文件>  →  输出候选 x 列表（显示坐标）
const fs = require("fs");
const path = require("path");
const HERE = __dirname;

// 极简 PNG 解码 → RGBA 像素（仅处理 8bit RGB/RGBA 非隔行 PNG）
const zlib = require("zlib");

function decodePng(buf) {
  let pos = 8; // 跳过签名
  let width = 0,
    height = 0,
    bitDepth = 0,
    colorType = 0;
  const idat = [];
  while (pos < buf.length) {
    const len = buf.readUInt32BE(pos);
    const type = buf.toString("ascii", pos + 4, pos + 8);
    const data = buf.subarray(pos + 8, pos + 8 + len);
    if (type === "IHDR") {
      width = data.readUInt32BE(0);
      height = data.readUInt32BE(4);
      bitDepth = data[8];
      colorType = data[9];
      if (data[12] !== 0) throw new Error("隔行 PNG 不支持");
    } else if (type === "IDAT") {
      idat.push(data);
    } else if (type === "IEND") {
      break;
    }
    pos += 12 + len;
  }
  if (bitDepth !== 8) throw new Error("仅支持 8bit PNG，实际 " + bitDepth);
  const channels = { 0: 1, 2: 3, 4: 2, 6: 4 }[colorType];
  if (!channels) throw new Error("不支持的 colorType " + colorType);
  const raw = zlib.inflateSync(Buffer.concat(idat));
  const stride = width * channels;
  const out = Buffer.alloc(height * stride);
  let rp = 0;
  for (let y = 0; y < height; y++) {
    const filter = raw[rp++];
    const row = raw.subarray(rp, rp + stride);
    rp += stride;
    const cur = out.subarray(y * stride, (y + 1) * stride);
    const prev = y > 0 ? out.subarray((y - 1) * stride, y * stride) : null;
    for (let x = 0; x < stride; x++) {
      const a = x >= channels ? cur[x - channels] : 0;
      const b = prev ? prev[x] : 0;
      const c = prev && x >= channels ? prev[x - channels] : 0;
      let v = row[x];
      switch (filter) {
        case 0:
          break;
        case 1:
          v = (v + a) & 0xff;
          break;
        case 2:
          v = (v + b) & 0xff;
          break;
        case 3:
          v = (v + ((a + b) >> 1)) & 0xff;
          break;
        case 4: {
          const p = a + b - c;
          const pa = Math.abs(p - a),
            pb = Math.abs(p - b),
            pc = Math.abs(p - c);
          const pr = pa <= pb && pa <= pc ? a : pb <= pc ? b : c;
          v = (v + pr) & 0xff;
          break;
        }
        default:
          throw new Error("未知 filter " + filter);
      }
      cur[x] = v;
    }
  }
  return { width, height, channels, data: out };
}

/** 逐列亮度（取指定行带的平均），返回 Float64Array */
function columnLuma(png, y0, y1) {
  const { width, height, channels, data } = png;
  const yA = Math.max(0, y0),
    yB = Math.min(height - 1, y1);
  const col = new Float64Array(width);
  for (let x = 0; x < width; x++) {
    let sum = 0,
      n = 0;
    for (let y = yA; y <= yB; y++) {
      const i = (y * width + x) * channels;
      sum += 0.299 * data[i] + 0.587 * data[i + 1] + 0.114 * data[i + 2];
      n++;
    }
    col[x] = n ? sum / n : 0;
  }
  return col;
}

/**
 * 找缺口候选 x（原始坐标系）。
 *
 * 精确依据（读 pkg/captcha/slider.go 得到）：
 *   `drawShadowOnBackgroundWithShape` 对形状内的像素做 `R/G/B ÷ 4`（**变暗到 25%**），
 *   随后 `drawGapBorder` 画半透明黑边框。
 *   ⇒ 缺口区域 = 一块**极暗**的形状。
 *
 * ⚠️ **不能用全局亮度阈值**：实测背景图本身明暗差异极大（同一行内亮度从 21 到 194），
 *    全图 65% 的像素低于「中位数 × 45%」→ 全局阈值把整张图判成暗区。
 *
 * ✅ **实测可行的判据（暗像素列密度法）**：
 *    统计每列中「亮度 < 30」的像素数 → 缺口形状的列（被 ÷4 压暗 + 黑边框）
 *    每列都有 30~89 个暗像素 → 呈现为**一段连续的密集高列**；
 *    而背景图的自然暗区每列只有 0~30 个。
 *    实测样例：缺口列 = x 189~222（其中 208~209 因拼图块箭头覆盖背景而仅 3~4 个，
 *    形成一个「凹陷」—— 该凹陷恰是缺口内部的特征，不影响左右边界识别）。
 *    ⇒ **xPos = 密集暗列段的起点**。
 *
 * 精度要求：服务端容差仅 **8px**（`internal/core/captcha.go:437`）。
 *    实测对照：粗略的「列均值分段法」给出 202（偏差 13px，**超容差**），
 *    暗像素列密度法给出 189（**命中**）。
 */
function findGapCandidates(png, pieceWidth) {
  const { width, height, channels, data } = png;
  const lum = (y, x) => {
    const i = (y * width + x) * channels;
    return 0.299 * data[i] + 0.587 * data[i + 1] + 0.114 * data[i + 2];
  };

  // 1) 每列的暗像素数（阈值 30：被 ÷4 压暗后的像素必然远低于此）
  const DARK = 30;
  const colDark = new Int32Array(width);
  for (let x = 0; x < width; x++) {
    let n = 0;
    for (let y = 0; y < height; y++) if (lum(y, x) < DARK) n++;
    colDark[x] = n;
  }
  // 2) 密集列阈值：形状列的暗像素数 ≥ 形状高度的 30%（实测形状高 70 → ≥21）。
  //    ⚠️ 阈值不能太高：初版用 40%（≥28）时，实测缺口段宽度只有 17~62（而非 ~70）
  //    —— 形状左/右缘有渐变像素未落入阴影 → 段被截短 → xPos 仍对但段不完整；
  //    降到 30% 后段宽接近 PieceWidth，排序判据才可靠。
  const minDense = Math.max(12, Math.round((pieceWidth || 70) * 0.3));
  const dense = [];
  for (let x = 0; x < width; x++) dense.push(colDark[x] >= minDense);

  // 3) 连续密集段（允许内部有 ≤3 列的凹陷 —— 拼图块箭头造成的）
  const segs = [];
  let start = -1;
  let gap = 0;
  for (let x = 0; x < width; x++) {
    if (dense[x]) {
      if (start < 0) start = x;
      gap = 0;
    } else if (start >= 0) {
      gap++;
      if (gap > 3) {
        segs.push({ start, end: x - gap });
        start = -1;
        gap = 0;
      }
    }
  }
  if (start >= 0) segs.push({ start, end: width - 1 });

  // 3b) ⚠️ **不合并相邻段**（初版合并 ≤12px 会把背景里的暗块与真缺口并成一个宽段，
  //     导致 xPos 取到背景块的位置 —— 实测表现为「识别值 220 → 实际缺口 162」）。
  //     真实的缺口形状本身就是**一个**连续密集段（内部凹陷 ≤3 列已由上面的 gap 容差吸收）。
  //
  // 3c) 候选排序：形状宽度最接近 PieceWidth 者优先（PieceWidth = 70/90/70 随难度），
  //     其次按平均暗度（缺口阴影是 ÷4 的极暗块）。
  const candidates = segs
    .filter((s) => s.end - s.start + 1 >= 8)
    .map((s) => {
      let sum = 0;
      for (let x = s.start; x <= s.end; x++) sum += colDark[x];
      const w = s.end - s.start + 1;
      return {
        xPos: s.start,
        start: s.start,
        end: s.end,
        width: w,
        avgDark: sum / w,
        widthScore: Math.abs(w - (pieceWidth || 70)),
      };
    })
    .sort((a, b) => a.widthScore - b.widthScore || b.avgDark - a.avgDark);

  return {
    candidates,
    median: Math.max(...colDark),
    threshold: minDense,
    colDarkProfile: Array.from(colDark),
  };
}

if (require.main === module) {
  const file = process.argv[2];
  const pieceWidth = parseInt(process.argv[3] || "80", 10);
  const buf = fs.readFileSync(file);
  const png = decodePng(buf);
  const { candidates, median, threshold } = findGapCandidates(png, pieceWidth);
  console.log(
    JSON.stringify(
      { width: png.width, height: png.height, median, threshold, candidates },
      null,
      1,
    ),
  );
}

module.exports = { decodePng, columnLuma, findGapCandidates };
