/**
 * Phase 156 / NOCODE-04：新建任务向导页面
 *
 * 路由注册方式（归属协议第 6 条）：**只**通过后端 sys_menu seed migration
 * （migration_256_nocode_wizard_menu.go）播种，
 * 前端**不改** componentLoader.tsx / DynamicRoutes.tsx —— 否则与 Phase 155
 * 在同一批文件上硬冲突。
 *
 * 页面职责：
 *   1. 承载 TaskWizard（三步）
 *   2. 把向导产出翻译成 POST /rpa/tasks 的请求体
 *   3. 处理三种来源的外部回调（AI 生成器 / 录制 / 模板）
 *   4. 保存后按运行方式决定是否立即执行
 */

import { useCallback, useState } from "react";
import { App, Button, Card, Layout, Space, Typography } from "antd";
import { ArrowLeftOutlined } from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import type { Action } from "@/types/rpa";
import { post } from "@/lib/api";
import { TaskWizard } from "./TaskWizard";
import type { WizardDraft } from "./payload";
import { toTaskPayload } from "./payload";
import { AIScriptEditor } from "../tasks/modals/AIScriptEditor";
import { useActionPermission } from "@/components/shared/useActionPermission";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
// W7 / UI-FIX-01：路由常量唯一事实来源。历史上这里三处写死
// `/operations/rpa/tasks` —— 该路径不存在任何路由（RPA 菜单由 sys_menu seed
// 播种，真实路由是 /vdi/rpa），写错会让 SPA 落到 catch-all → dashboard。
import { RPA_ROUTES } from "../routes";

const { Content } = Layout;
const { Title, Text } = Typography;

export default function TaskWizardPage() {
  const { message } = App.useApp();
  const navigate = useNavigate();

  // W7.3 / RBAC 门控：向导页是「新建任务」的写入口，必须接入三态门控。
  // gating 覆盖率门禁实测暴露本页未覆盖（v1.56 Phase 156 新增页面时漏铺）。
  const permGate = useActionPermission();
  const canAdd = permGate.decide(PERMISSION_CODES.rpaTask.add).visible;

  const [aiOpen, setAiOpen] = useState(false);
  const [pendingActions, setPendingActions] = useState<Action[] | undefined>();
  const [pendingDescription, setPendingDescription] = useState<string | undefined>();

  /**
   * handleSave 保存任务；runMode=immediate 时顺带触发一次执行。
   *
   * 「立即运行」的语义是「保存后立刻跑一次」而不是「不保存就跑」——
   * 用户能立刻在任务列表里看到它，也能看到执行结果。
   */
  const handleSave = useCallback(
    async (draft: WizardDraft) => {
      const payload = toTaskPayload(draft);
      const created = await post<{ id?: string }>("/rpa/tasks", payload);
      const taskId = created?.data?.id;

      if (draft.runMode === "immediate" && taskId) {
        try {
          await post(`/rpa/tasks/${taskId}/execute`, {});
          message.success("任务已保存，并已开始运行");
        } catch {
          // 执行失败不掩盖「保存成功」这个事实
          message.warning("任务已保存，但立即运行没有成功，可到任务列表里手动执行");
        }
      } else {
        message.success("任务已保存");
      }
      navigate(RPA_ROUTES.taskList);
    },
    [message, navigate]
  );

  /** 无 ops:rpa:task:add 权限时拦住保存（门控在前端也必须生效，不能只靠后端 403）。 */
  const guardedSave = useCallback(
    async (draft: WizardDraft) => {
      if (!canAdd) {
        message.warning("你没有新建自动化任务的权限，请联系管理员开通");
        return;
      }
      await handleSave(draft);
    },
    [canAdd, handleSave, message]
  );

  /** AI 生成完成 → 回填到向导并重新挂载（用 key 触发 state 重置） */
  const handleAIConfirm = useCallback(
    (actions: Action[]) => {
      setPendingActions(actions);
      setAiOpen(false);
      message.success(`已生成 ${actions.length} 个步骤，请核对`);
    },
    [message]
  );

  return (
    <Layout style={{ background: "transparent" }}>
      <Content style={{ padding: 0, background: "transparent" }}>
        <Card>
          <Space style={{ width: "100%", justifyContent: "space-between" }}>
            <div>
              <Title level={4} style={{ margin: 0 }}>
                新建自动化任务
              </Title>
              <Text type="secondary">三步就能让电脑替你重复做这些事，不需要懂任何技术。</Text>
            </div>
            <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(RPA_ROUTES.taskList)}>
              返回任务列表
            </Button>
          </Space>
        </Card>

        <Card style={{ marginTop: 16 }}>
          <TaskWizard
            key={`${pendingActions?.length ?? 0}-${pendingActions?.[0]?.id ?? "init"}`}
            onSave={guardedSave}
            onCancel={() => navigate(RPA_ROUTES.taskList)}
            onOpenAIGenerator={() => setAiOpen(true)}
            onOpenRecorder={() => {
              // 录制入口：当前通过任务编辑器的「一键转录制」与 live 画面提供；
              // 这里给出明确指引，避免「点了没反应」。
              message.info("请在「实时画面」里点「开始录制」，录完会自动回到这里");
            }}
            initialActions={pendingActions}
            initialDescription={pendingDescription}
          />
        </Card>

        {/* NOCODE-03：AI 生成器叠加在向导之上；不可用时降级为录制指引 */}
        <AIScriptEditor
          open={aiOpen}
          onClose={() => setAiOpen(false)}
          onConfirm={handleAIConfirm}
          onFallbackToRecord={() => {
            setAiOpen(false);
            setPendingDescription("");
            message.info("请改用「录一段操作」：在实时画面里开始录制，录完自动生成步骤");
          }}
        />
      </Content>
    </Layout>
  );
}

// 保持与既有页面一致：默认导出（componentLoader 依赖 default）
export { TaskWizardPage };
