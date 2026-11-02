package portcollection

import (
	"fmt"
	"strings"

	"github.com/xingran-next/xingran-go-backend/internal/models"
)

// Phase 129 真机 UAT 修复（2026-11-01 华为 S8700 + 锐捷 RS8607E 实测）：
// 厂商/型号能力矩阵 —— 让不支持的动作返回明确错误，而不是下发必失败命令。
//
// 真机事实（华为 S8700 10.62.25.253，接口 GE1/0/0，设备 `?` 探测）：
//   1. user-bind 仅系统视图可用（接口视图 Unrecognized）→ 华为 port_binding
//      命令渲染为无 interface 前缀的系统视图序列（RequiresInterfaceView=false）
//   2. `port-security ?` 子命令仅 aging-time/enable/mac-address/protect-action
//      → S8700 无 max-mac-num → portsec_max_mac 不支持
//   3. `port ?` 子命令无 trunk/hybrid → S8700 无 trunk/hybrid 关键字
//      （port default vlan / port link-type 是正确关键字，既有动作不受影响）
//
// 设计：
//   - 厂商基线（vendor baseline）：该厂商各动作默认支持级别
//   - 型号规则（model rules）：按 NetworkDevice.Model 字符串匹配叠加覆盖
//     （如 "S87" 匹配 S8700 系列）；未匹配型号沿用厂商基线（保守兼容）
//   - 校验入口 CheckActionSupport：写路径与 dry-run 前统一调用，不支持动作
//     返回 ErrActionNotSupported（handler 翻译 400 + reason 含真机依据）

// ActionSupport 动作支持级别。
type ActionSupport int

const (
	// ActionSupported 支持（常规接口视图执行）。
	ActionSupported ActionSupport = iota
	// ActionSystemView 需在系统视图执行（命令序列不带 interface 前缀，
	// executeWrite 不追加 view 退出命令）。
	ActionSystemView
	// ActionUnsupported 该厂商/型号不支持（下发必失败，应显式拒绝）。
	ActionUnsupported
)

// ErrActionNotSupported 能力矩阵拒绝（%w 链携带 vendor/model/action/reason）。
var ErrActionNotSupported = fmt.Errorf("portcollection: action not supported on this device")

// capabilityEntry 单厂商能力描述。
type capabilityEntry struct {
	// baseline 厂商基线：action → 级别（缺省 ActionSupported）
	baseline map[PortAction]ActionSupport
	// modelRules 型号覆盖规则：model 子串（大小写不敏感）→ 覆盖表
	modelRules []modelCapabilityRule
	// vendorUnsupported 厂商级「不支持」的动作集合（action → 原因）。
	//
	// ★ 为什么需要它（2026-11-02 Cisco 裁定）：型号级规则靠 model 子串匹配，
	//   而「整个厂商都不支持某动作」与型号无关 ⇒ 无法用 modelRules 表达
	//   （matchModelSubstr 为空数组时永不匹配；model 为空时整组规则不生效）。
	//   之前 Cisco 因为「矩阵里没有条目」而落到「默认放行」—— 那正是
	//   **把推测当默认**（默认支持 ≠ 有证据支持）。
	vendorUnsupported map[PortAction]string
}

// modelCapabilityRule 型号级能力覆盖。
type modelCapabilityRule struct {
	// matchModelSubstr model 字符串包含任一子串即命中（大小写不敏感）
	matchModelSubstr []string
	// unsupported 命中后不支持的动作集合
	unsupported map[PortAction]string // action → reason（真机依据）
	// supported 命中后「真机验证支持」的动作集合（action → 真机依据）。
	//
	// 为什么需要显式的 supported（2026-11-01 锐捷 RS8607E 真机复测）：
	// 华为 S8700 与锐捷 RS8607E 在 portsec_static_bind / portsec_max_mac 上
	// 结论完全相反 —— 前者 Unsupported，后者实测无错误。若矩阵只记「拒绝」，
	// 「支持」就是隐式默认值，后来者容易把锐捷按华为一刀切掉。
	// 显式记录支持结论 + 真机依据，测试锁定两家结论相反。
	supported map[PortAction]string
}

// ciscoUnsupportedReason Cisco 不支持 trunk/hybrid 动作的原因（诚实标注依据）。
//
// ⚠️ 依据是**「未实现 + 未取证」**，不是「真机证明不支持」—— 无设备时不伪造真机结论。
const ciscoUnsupportedReason = "Cisco 当前不支持该端口写操作：渲染层未实现 trunk/hybrid 渲染，" +
	"且无真机取证依据（2026-11-02 裁定：矩阵显式标记 Unsupported，不以推测冒充真机结论）"

// vendorCapabilities 厂商能力矩阵（真机事实唯一真相源，测试锁定）。
var vendorCapabilities = map[models.DeviceVendor]capabilityEntry{
	models.VendorHuawei: {
		baseline: map[PortAction]ActionSupport{
			// 真机事实 #1：user-bind 是系统视图命令（接口视图 Unrecognized）
			ActionPortBinding: ActionSystemView,
		},
		modelRules: []modelCapabilityRule{
			{
				// 真机事实 #2/#3：S8700（S87xx）无 port trunk/hybrid 关键字、
				// port-security 无 max-mac-num 子命令
				matchModelSubstr: []string{"s87", "s8700"},
				unsupported: map[PortAction]string{
					TrunkSetNative:    "S8700 接口视图 port ? 无 trunk 子命令（2026-11-01 真机 ? 探测）",
					TrunkSetAllowed:   "S8700 接口视图 port ? 无 trunk 子命令（2026-11-01 真机 ? 探测）",
					TrunkSetHybrid:    "S8700 接口视图 port ? 无 hybrid 子命令（2026-11-01 真机 ? 探测）",
					PortSecMaxMac:     "S8700 port-security ? 无 max-mac-num 子命令（2026-11-01 真机 ? 探测）",
					PortSecStaticBind: "S8700 port-security mac-address ? 仅 H-H-H/sticky 无 static；sticky <MAC> 报 Wrong parameter ⇒ 不支持静态 MAC 绑定（2026-11-01 真机探测）",
				},
				// 同一次真机复测里「支持」的两项也要显式记下来：sticky 开关式可用、
				// port_binding 仅 IP-only 变体可用（IP+MAC 组合由参数级规则拒绝）。
				// 记下来是为了与锐捷 RS8607E 的 supported 形成对照，避免误以为
				// 「华为整个 port-security 家族都不支持」。
				supported: map[PortAction]string{
					PortSecSticky:     "S8700 真机（2026-11-01）：port-security mac-address sticky 实测无错误 ⇒ sticky 仅作开关式使用（不支持带 MAC）",
					ActionPortBinding: "S8700 真机（2026-11-01）：user-bind static ip-address 10.62.25.5（IP-only）实测合法 ⇒ 仅 IP-only 变体可用，IP+MAC 组合报 Wrong parameter",
				},
			},
		},
	},
	// 华为 static_bind 补充（2026-11-01 接口视图探测）：port-security
	// mac-address sticky ? 只接受 [H-H-H | sticky]，无 vlan 参数 —— S8700 的
	// "静态 MAC 绑定"只能经 sticky 表达（static 子关键字该型号不存在）。
	// 渲染层已按此去掉 vlan 参数；本矩阵不需拒绝（命令合法，仅参数形状不同）。
	//
	// H3C：真机证据不足，暂按默认支持（S8700 之外的华为 CE/S57 系列
	// 按公允语法支持 trunk/hybrid/max-mac-num）—— ⚠️ 这是**未取证但被当作默认**，
	// 取证计划见 .planning/phases/130-device-evidence/130-EVIDENCE-PLAN-NO-DEVICE.md。
	//
	// Cisco（2026-11-02 裁定）：**显式标 Unsupported**，不再依赖「矩阵里没有条目
	// ⇒ 默认放行」—— 那正是把推测当默认。依据诚实标注：**渲染层未实现 + 无真机取证依据**，
	// 不伪造真机结论。
	models.VendorCisco: {
		vendorUnsupported: map[PortAction]string{
			TrunkSetNative:  ciscoUnsupportedReason,
			TrunkSetAllowed: ciscoUnsupportedReason,
			TrunkSetHybrid:  ciscoUnsupportedReason,
		},
	},
	// 锐捷 hybrid：2026-11-02 真机 `?` 帮助确认设备支持，此前渲染层未注册
	// （RenderCommand → ErrUnknownAction）属功能缺口，本次已补齐渲染，
	// 矩阵显式记 supported（见下方锐捷条目）。

	// 锐捷 RS8607E 真机（2026-11-01 锐捷 12 动作复测，走产品 scrapli 链路）：
	// static_bind / max_mac / sticky / port_binding 全部实测无错误 ——
	// 与华为 S8700 的 static_bind、max_mac 结论相反（两家不可一刀切）。
	models.VendorRuijie: {
		modelRules: []modelCapabilityRule{
			{
				// ★ 2026-11-02 补漏配：库里真实存在三种命名 —— RS8607E / RG8607E /
				//   RG-S8607E-iSeries。第三次真机取证确认 **RG8607E 与 RS8607E 的
				//   hybrid 关键字形态完全一致**（同一渲染可覆盖两条产品线），
				//   但此前只匹配 rs8607 ⇒ RG8607E 设备会被判为"型号不支持"而误拒。
				matchModelSubstr: []string{"rs8607", "rg8607", "rg-s8607"},
				supported: map[PortAction]string{
					PortSecStaticBind: "锐捷 RS8607E 真机（2026-11-01）：switchport port-security mac-address <mac> vlan <vlan> 实测无错误 ⇒ 支持静态 MAC 绑定（华为 S8700 同动作 Unsupported，两家结论相反）",
					PortSecMaxMac:     "锐捷 RS8607E 真机（2026-11-01）：switchport port-security maximum 5 实测无错误（参数范围 <1-128>）⇒ 支持 MAC 数上限（华为 S8700 无 max-mac-num 子命令）",
					PortSecSticky:     "锐捷 RS8607E 真机（2026-11-01）：switchport port-security mac-address sticky 实测无错误 ⇒ 支持 sticky MAC 模式",
					ActionPortBinding: "锐捷 RS8607E 真机（2026-11-01）：switchport port-security binding <IP> <MAC> 实测无错误 ⇒ 支持 IP+MAC 组合绑定（IP 在前，与华为 S8700 IP+MAC Unsupported 相反）",
					// 2026-11-02 追加（渲染缺口补齐，非 uotester 既有内容）；
					// 二次取证（`?` 帮助）：allowed 必须带 tagged/untagged 关键字，native 是单值 VID；
					// 三次取证（真机实跑，RS8607E + RG8607E 形态一致）修正：
					//   tagged 必带 only、untagged 不能带 only、还原只认模式回退。
					// ★ 2026-11-02 第六轮实跑定稿：native 形态**原样有效、无 only 要求**（不是第二个 P0）。
					TrunkSetNative: "锐捷真机（2026-11-02 六轮实跑，RS8607E / RG8607E）：" +
						"switchport trunk native vlan <id> 形态**原样有效、无 only 要求**（命令未被拒绝，落盘形态原样未被改写）；" +
						"与 allowed 列表是**独立维度**（设 native 不影响 allowed 行）；" +
						"还原 switchport mode access 对 native 也有效（一次清掉）；" +
						"⚠️ 本次执行前 native 行已是 3990（设备原有配置）⇒ " +
						"「命令是否真的改写了 native 值」未单独验证；" +
						"本结论仅限 RS8607E / RG8607E，**不外推其它型号**",
					// ⚠️ 历史：TrunkSetNative 曾标「未取证」（2026-11-02 待实测）——
					//   实测后已确计（见上方 supported 依据）⇒ 已改为有证据的 supported。
					TrunkSetAllowed: "锐捷真机（2026-11-02 五轮实跑定稿，RS8607E 与 RG8607E 形态一致）：" +
						"replace ⇒ switchport trunk allowed vlan only <list>（★ only 本身即全量语义：实测 only 3990 直接覆盖原列表）→" +
						"add ⇒ switchport trunk allowed vlan add <list>（并集/叠加：only 3990 → add 3991 ⇒ only 3990-3991）→" +
						"remove ⇒ switchport trunk allowed vlan remove <list>（移除：only 3990-3991 → remove 3991 ⇒ only 3990）；" +
						"★ 无关键字形态 switchport trunk allowed vlan <list> 真机 Invalid input（**非法命令**）；" +
						"另：锐捷 trunk allowed **不需预建 VLAN**（实测不自动创建 VLAN 3990/3991 —— 与 hybrid 不同，hybrid 会自动建）；" +
						"本结论仅限 RS8607E / RG8607E，**不外推其它型号**",
					TrunkSetHybrid: "锐捷真机（2026-11-02 三次取证，RS8607E 与 RG8607E 形态一致）：" +
						"switchport mode hybrid（前置）→" +
						"switchport hybrid allowed vlan only tagged <list>（★ 必带 only：缺 only 无错误但配置不落地）→" +
						"switchport hybrid allowed vlan untagged <list>（★ 不能带 only：only untagged 被设备拒绝）→" +
						"switchport hybrid native vlan <id>（单值 VID，与两个列表解耦）；" +
						"还原只认模式回退 switchport mode access（逐条 no switchport hybrid ... 被设备拒绝；" +
						"还原失败 = 口留在 hybrid 模式 = 转发行为改变）；" +
						"另：端口启用 802.1X（dot1x port-control auto）时设备拒绝切 hybrid" +
						"（Can not operate on interface ... Conflict with IEEE 802.1X.）⇒ 由 portwrite 预检层拦截",
				},
			},
		},
	},
}

// RequiresInterfaceView 该动作命令序列是否需要 interface 前缀视图。
//
// false 时 executeWrite 不追加 view 退出命令（命令直接在系统视图执行，
// 追加 exit 会把会话掉出配置视图破坏 scrapli priv 跟踪）。
func RequiresInterfaceView(vendor models.DeviceVendor, action PortAction) bool {
	return lookupSupport(vendor, "", action) != ActionSystemView
}

// matchModelRules 返回 (vendor, model) 命中的型号级规则（型号子串匹配，大小写不敏感）。
//
// 抽出来是为了 lookupSupport / unsupportedReason / SupportEvidence 三处共用同一套
// 匹配语义（避免三份拷贝各自漂移）。vendor 不在矩阵内或 model 为空 → 无型号级规则
// （只按厂商基线处理）。
func matchModelRules(vendor models.DeviceVendor, model string) []modelCapabilityRule {
	entry, ok := vendorCapabilities[vendor]
	if !ok || model == "" {
		return nil
	}
	lower := strings.ToLower(model)
	var matched []modelCapabilityRule
	for _, rule := range entry.modelRules {
		for _, substr := range rule.matchModelSubstr {
			if strings.Contains(lower, substr) {
				matched = append(matched, rule)
				break
			}
		}
	}
	return matched
}

// lookupSupport 解析 (vendor, model, action) → 支持级别（model 规则优先于基线）。
func lookupSupport(vendor models.DeviceVendor, model string, action PortAction) ActionSupport {
	entry, ok := vendorCapabilities[vendor]
	if !ok {
		return ActionSupported
	}
	matched := matchModelRules(vendor, model)
	// 型号级「不支持」优先于一切（含型号级显式支持）
	for _, rule := range matched {
		if _, banned := rule.unsupported[action]; banned {
			return ActionUnsupported
		}
	}
	// 型号级显式「真机验证支持」：不得被厂商基线降级为 Unsupported，
	// 但基线的系统视图要求（ActionSystemView）是渲染路径约束，予以保留。
	for _, rule := range matched {
		if _, ok := rule.supported[action]; ok {
			if lvl, ok := entry.baseline[action]; ok && lvl == ActionSystemView {
				return ActionSystemView
			}
			return ActionSupported
		}
	}
	// 厂商级「不支持」（与型号无关）：如 Cisco 的 trunk/hybrid（渲染层未实现 +
	// 无真机取证依据）。放在型号级 supported 之后 —— 若将来某型号真机验证支持，
	// 型号级证据优先。
	if _, banned := entry.vendorUnsupported[action]; banned {
		return ActionUnsupported
	}
	if lvl, ok := entry.baseline[action]; ok {
		return lvl
	}
	return ActionSupported
}

// unsupportedReason 返回型号级不支持原因（无则空串）。
func unsupportedReason(vendor models.DeviceVendor, model string, action PortAction) string {
	for _, rule := range matchModelRules(vendor, model) {
		if reason, banned := rule.unsupported[action]; banned {
			return reason
		}
	}
	if entry, ok := vendorCapabilities[vendor]; ok {
		if reason, banned := entry.vendorUnsupported[action]; banned {
			return reason
		}
	}
	return ""
}

// SupportEvidence 返回 (vendor, model, action) 的真机依据文本。
//
// 命中型号级 unsupported → 拒绝依据（真机实测不支持的原因）；
// 命中型号级 supported   → 支持依据（真机实测可用的命令形态）；
// 未命中型号级规则        → 空串（无型号级真机证据，按厂商基线处理）。
//
// 用途：把「为什么这家支持、那家不支持」显式暴露给调用方（handler 错误文案 /
// 审计 / 前端提示），也是矩阵按厂商区分的可读出口。
func SupportEvidence(vendor models.DeviceVendor, model string, action PortAction) string {
	for _, rule := range matchModelRules(vendor, model) {
		if reason, banned := rule.unsupported[action]; banned {
			return reason
		}
		if evidence, ok := rule.supported[action]; ok {
			return evidence
		}
	}
	if entry, ok := vendorCapabilities[vendor]; ok {
		if reason, banned := entry.vendorUnsupported[action]; banned {
			return reason
		}
	}
	return ""
}

// paramComboRule 参数级组合拒绝规则（action 内部参数形状差异，如
// 华为 S8700 user-bind static 的 IP+MAC 组合不被接受而 IP-only 合法）。
type paramComboRule struct {
	action PortAction
	// matchModelSubstr 型号子串匹配（大小写不敏感；真机事实只锚定 S8700）
	matchModelSubstr []string
	// reject 当 params 满足谓词时拒绝
	reject func(macAddress string) bool
	reason string
}

// vendorParamComboRules 参数级组合规则（真机事实锚定；仅命中型号生效）。
var vendorParamComboRules = map[models.DeviceVendor][]paramComboRule{
	models.VendorHuawei: {
		{
			action:           ActionPortBinding,
			matchModelSubstr: []string{"s87", "s8700"},
			// S8700：user-bind static ip-address ? 仅接受 X.X.X.X，
			// 跟随 mac-address 报 Wrong parameter（2026-11-01 真机探测）
			reject: func(mac string) bool { return mac != "" },
			reason: "S8700 user-bind static ip-address ? 仅接受 X.X.X.X，IP+MAC 组合绑定报 Wrong parameter（2026-11-01 真机探测）；IP-only 绑定可用",
		},
	},
}

// CheckActionSupportParams 参数感知校验：在 CheckActionSupport 基础上叠加
// 参数组合规则（macAddress 取 PortBinding 的 MAC 字段；其余动作忽略）。
// 写路径与 dry-run 统一调用本入口。
func CheckActionSupportParams(vendor models.DeviceVendor, model string, action PortAction, macAddress string) error {
	if err := CheckActionSupport(vendor, model, action); err != nil {
		return err
	}
	lowerModel := strings.ToLower(model)
	for _, rule := range vendorParamComboRules[vendor] {
		if rule.action != action || !rule.reject(macAddress) {
			continue
		}
		matched := false
		for _, substr := range rule.matchModelSubstr {
			if strings.Contains(lowerModel, substr) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		return fmt.Errorf("%w: vendor=%s model=%q action=%s — %s",
			ErrActionNotSupported, vendor, model, action, rule.reason)
	}
	return nil
}

// CheckActionSupport 能力矩阵校验入口（写路径与 dry-run 统一调用）。
//
// model 为设备型号（models.NetworkDevice.Model，可空——空则仅厂商基线生效）。
// 不支持返回 ErrActionNotSupported 包装错误（reason 含真机依据）。
func CheckActionSupport(vendor models.DeviceVendor, model string, action PortAction) error {
	switch lookupSupport(vendor, model, action) {
	case ActionUnsupported:
		reason := unsupportedReason(vendor, model, action)
		if reason == "" {
			reason = "该厂商不支持此动作"
		}
		return fmt.Errorf("%w: vendor=%s model=%q action=%s — %s",
			ErrActionNotSupported, vendor, model, action, reason)
	default:
		return nil
	}
}
