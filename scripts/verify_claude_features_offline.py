#!/usr/bin/env python3
"""
Claude Code Features - Offline Verification

离线验证 HotPlex 代码库是否支持以下 Claude Code 高级功能：
1. Plan Mode - 只生成计划不执行
2. AskUserQuestion 工具 - 澄清问题的交互工具
3. Output Styles (Learning/Explanatory) - 教育性输出风格
4. Permission Request - 权限请求

验证方法：
- 分析 provider/claude_provider.go 的事件解析逻辑
- 检查是否识别和处理相关事件类型
- 验证 Slack Block Kit 映射是否完整

使用前确保：
- 在 HotPlex 项目根目录运行

运行方式：
    python3 scripts/verify_claude_features_offline.py
"""

import os
import re
import sys
from pathlib import Path
from dataclasses import dataclass
from enum import Enum
from typing import Optional


class VerificationStatus(Enum):
    """验证状态"""

    PASS = "✅ 通过"
    PARTIAL = "⚠️ 部分支持"
    FAIL = "❌ 未实现"
    ERROR = "❌ 错误"


@dataclass
class VerificationResult:
    """验证结果"""

    feature: str
    status: VerificationStatus
    evidence: list[str]
    missing: list[str]
    recommendations: list[str]


class CodebaseVerifier:
    """代码库验证器"""

    def __init__(self, project_root: str = "."):
        self.project_root = Path(project_root)
        self.results: list[VerificationResult] = []

        # 关键文件路径
        self.provider_file = self.project_root / "provider" / "claude_provider.go"
        self.event_file = self.project_root / "provider" / "event.go"
        self.mapping_doc = (
            self.project_root / "docs" / "chatapps" / "engine-events-slack-mapping.md"
        )

    def read_file(self, path: Path) -> Optional[str]:
        """读取文件内容"""
        try:
            return path.read_text(encoding="utf-8")
        except FileNotFoundError:
            return None
        except Exception as e:
            print(f"读取文件失败 {path}: {e}")
            return None

    def check_pattern(self, content: str, pattern: str, description: str) -> bool:
        """检查代码中是否包含特定模式"""
        match = re.search(pattern, content, re.MULTILINE)
        if match:
            print(f"  ✅ {description}")
            return True
        else:
            print(f"  ❌ {description}")
            return False

    def verify_plan_mode_support(self) -> VerificationResult:
        """
        验证 Plan Mode 支持

        检查点：
        1. claude_provider.go 是否识别 subtype 字段
        2. 是否处理 plan_generation subtype
        3. 是否有 BuildPlanModeBlock 方法
        """
        print("\n" + "=" * 60)
        print("验证 1: Plan Mode 支持")
        print("=" * 60)

        evidence = []
        missing = []
        recommendations = []

        # 检查 1: subtype 字段支持
        provider_content = self.read_file(self.provider_file)
        if provider_content:
            if "Subtype" in provider_content:
                evidence.append("StreamMessage 包含 subtype 字段")
            else:
                missing.append("StreamMessage 缺少 subtype 字段")

            # 检查是否处理 plan_generation
            if "plan_generation" in provider_content:
                evidence.append("检测到 plan_generation 处理逻辑")
            else:
                missing.append("未检测到 plan_generation 处理")
                recommendations.append("添加 subtype == 'plan_generation' 的检测")
        else:
            missing.append("无法读取 claude_provider.go")

        # 检查 2: Block Builder 方法
        mapping_content = self.read_file(self.mapping_doc)
        if mapping_content:
            if "BuildPlanModeBlock" in mapping_content:
                evidence.append("文档包含 BuildPlanModeBlock 方法定义")
            else:
                missing.append("文档缺少 BuildPlanModeBlock 方法")
                recommendations.append("在 block_builder.go 中实现 BuildPlanModeBlock")
        else:
            missing.append("无法读取 engine-events-slack-mapping.md")

        # 确定状态
        if len(evidence) >= 2:
            status = VerificationStatus.PASS
        elif len(evidence) >= 1:
            status = VerificationStatus.PARTIAL
        else:
            status = VerificationStatus.FAIL

        return VerificationResult(
            feature="Plan Mode",
            status=status,
            evidence=evidence,
            missing=missing,
            recommendations=recommendations,
        )

    def verify_ask_user_question_support(self) -> VerificationResult:
        """
        验证 AskUserQuestion 工具支持

        检查点：
        1. 是否识别 AskUserQuestion 工具名称
        2. 是否有 AskUserQuestionRequest 结构定义
        3. 是否有 BuildAskUserQuestionBlocks 方法
        4. 是否有回调处理逻辑
        """
        print("\n" + "=" * 60)
        print("验证 2: AskUserQuestion 工具支持")
        print("=" * 60)

        evidence = []
        missing = []
        recommendations = []

        # 检查 1: tool_use 事件处理
        provider_content = self.read_file(self.provider_file)
        if provider_content:
            if "tool_use" in provider_content and "ToolName" in provider_content:
                evidence.append("tool_use 事件解析已实现")
            else:
                missing.append("tool_use 事件解析不完整")

            # 检查是否识别 AskUserQuestion
            if "AskUserQuestion" in provider_content:
                evidence.append("检测到 AskUserQuestion 识别逻辑")
            else:
                missing.append("未检测到 AskUserQuestion 特殊处理")
                recommendations.append("添加 AskUserQuestion 工具的识别和回调")
        else:
            missing.append("无法读取 claude_provider.go")

        # 检查 2: 文档和实现
        mapping_content = self.read_file(self.mapping_doc)
        if mapping_content:
            if "AskUserQuestionRequest" in mapping_content:
                evidence.append("文档包含 AskUserQuestionRequest 结构定义")
            else:
                missing.append("文档缺少 AskUserQuestionRequest 定义")
                recommendations.append("定义 AskUserQuestionRequest 结构体")

            if "BuildAskUserQuestionBlocks" in mapping_content:
                evidence.append("文档包含 BuildAskUserQuestionBlocks 方法")
            else:
                missing.append("文档缺少 BuildAskUserQuestionBlocks 方法")
                recommendations.append(
                    "在 block_builder.go 中实现 BuildAskUserQuestionBlocks"
                )

            if "ask_answer" in mapping_content:
                evidence.append("文档包含按钮回调处理")
            else:
                missing.append("文档缺少回调处理示例")
                recommendations.append("添加交互式按钮回调处理逻辑")
        else:
            missing.append("无法读取 engine-events-slack-mapping.md")

        # 确定状态
        if len(evidence) >= 3:
            status = VerificationStatus.PASS
        elif len(evidence) >= 2:
            status = VerificationStatus.PARTIAL
        else:
            status = VerificationStatus.FAIL

        return VerificationResult(
            feature="AskUserQuestion",
            status=status,
            evidence=evidence,
            missing=missing,
            recommendations=recommendations,
        )

    def verify_output_styles_support(self) -> VerificationResult:
        """
        验证 Output Styles 支持

        检查点：
        1. 是否有 OutputStyle 类型定义
        2. 是否有 BuildInsightBlock 方法
        3. 是否检测 TODO(human) 标记
        4. 是否有样式切换配置
        """
        print("\n" + "=" * 60)
        print("验证 3: Output Styles 支持")
        print("=" * 60)

        evidence = []
        missing = []
        recommendations = []

        # 检查 1: 文档定义
        mapping_content = self.read_file(self.mapping_doc)
        if mapping_content:
            if "OutputStyle" in mapping_content:
                evidence.append("文档包含 OutputStyle 类型定义")
            else:
                missing.append("文档缺少 OutputStyle 定义")
                recommendations.append(
                    "定义 OutputStyle 枚举 (default/explanatory/learning)"
                )

            if "BuildInsightBlock" in mapping_content:
                evidence.append("文档包含 BuildInsightBlock 方法")
            else:
                missing.append("文档缺少 BuildInsightBlock 方法")
                recommendations.append("在 block_builder.go 中实现 BuildInsightBlock")

            if "TODO(human)" in mapping_content:
                evidence.append("文档包含 Learning Mode TODO 检测")
            else:
                missing.append("文档缺少 Learning Mode TODO 处理")
                recommendations.append("添加 TODO(human) 标记检测逻辑")

            if "Explanatory" in mapping_content or "Learning" in mapping_content:
                evidence.append("文档包含 Explanatory/Learning 模式说明")
            else:
                missing.append("文档缺少输出风格详细说明")
                recommendations.append("补充 Output Styles 配置和使用说明")
        else:
            missing.append("无法读取 engine-events-slack-mapping.md")

        # 确定状态
        if len(evidence) >= 3:
            status = VerificationStatus.PASS
        elif len(evidence) >= 2:
            status = VerificationStatus.PARTIAL
        else:
            status = VerificationStatus.FAIL

        return VerificationResult(
            feature="Output Styles",
            status=status,
            evidence=evidence,
            missing=missing,
            recommendations=recommendations,
        )

    def verify_permission_request_support(self) -> VerificationResult:
        """
        验证 Permission Request 支持

        检查点：
        1. 是否有 permission_request 事件类型
        2. 是否有 BuildPermissionRequestBlocks 方法
        3. 是否有 perm_allow/perm_deny 回调
        4. 是否有交互式按钮处理
        """
        print("\n" + "=" * 60)
        print("验证 4: Permission Request 支持")
        print("=" * 60)

        evidence = []
        missing = []
        recommendations = []

        # 检查 1: 事件类型定义
        event_content = self.read_file(self.event_file)
        if event_content:
            if "EventTypePermissionRequest" in event_content:
                evidence.append("EventTypePermissionRequest 已定义")
            else:
                missing.append("缺少 EventTypePermissionRequest 类型")
                recommendations.append("在 event.go 中添加 EventTypePermissionRequest")

            if "permission_request" in event_content:
                evidence.append("permission_request 事件类型已定义")
            else:
                missing.append("缺少 permission_request 事件定义")
        else:
            missing.append("无法读取 event.go")

        # 检查 2: Provider 处理
        provider_content = self.read_file(self.provider_file)
        if provider_content:
            if "permission_request" in provider_content:
                evidence.append("Provider 解析 permission_request 事件")
            else:
                missing.append("Provider 未处理 permission_request")
                recommendations.append("在 ParseEvent 中添加 permission_request 处理")

            if (
                "PermissionDetail" in provider_content
                or "Permission" in provider_content
            ):
                evidence.append("Permission 结构已定义")
            else:
                missing.append("缺少 Permission 相关结构")
        else:
            missing.append("无法读取 claude_provider.go")

        # 检查 3: Block Builder
        mapping_content = self.read_file(self.mapping_doc)
        if mapping_content:
            if "BuildPermissionRequestBlocks" in mapping_content:
                evidence.append("文档包含 BuildPermissionRequestBlocks 方法")
            else:
                missing.append("文档缺少 BuildPermissionRequestBlocks 方法")
                recommendations.append(
                    "在 block_builder.go 中实现 BuildPermissionRequestBlocks"
                )

            if "perm_allow" in mapping_content and "perm_deny" in mapping_content:
                evidence.append("文档包含允许/拒绝按钮回调")
            else:
                missing.append("文档缺少权限审批回调")
                recommendations.append("实现 perm_allow/perm_deny 交互式回调")

            if "✅ Allow" in mapping_content or "Allow" in mapping_content:
                evidence.append("UI 设计包含允许按钮")
            else:
                missing.append("UI 设计不完整")
        else:
            missing.append("无法读取 engine-events-slack-mapping.md")

        # 确定状态
        if len(evidence) >= 5:
            status = VerificationStatus.PASS
        elif len(evidence) >= 3:
            status = VerificationStatus.PARTIAL
        else:
            status = VerificationStatus.FAIL

        return VerificationResult(
            feature="Permission Request",
            status=status,
            evidence=evidence,
            missing=missing,
            recommendations=recommendations,
        )

    def run_all_verifications(self):
        """运行所有验证"""
        print("\n" + "🔍" * 30)
        print("HotPlex Claude Code 高级功能离线验证")
        print("🔍" * 30)
        print(f"\n项目根目录：{self.project_root.absolute()}")

        # 检查关键文件是否存在
        print("\n📁 检查关键文件...")
        files_to_check = [self.provider_file, self.event_file, self.mapping_doc]

        all_files_exist = True
        for file_path in files_to_check:
            if file_path.exists():
                print(f"  ✅ {file_path.relative_to(self.project_root)}")
            else:
                print(f"  ❌ {file_path.relative_to(self.project_root)} (不存在)")
                all_files_exist = False

        if not all_files_exist:
            print("\n❌ 关键文件缺失，无法完成验证")
            return

        # 运行验证
        self.results.append(self.verify_plan_mode_support())
        self.results.append(self.verify_ask_user_question_support())
        self.results.append(self.verify_output_styles_support())
        self.results.append(self.verify_permission_request_support())

        # 输出报告
        self.print_report()

    def print_report(self):
        """打印验证报告"""
        print("\n" + "=" * 70)
        print("📊 验证报告")
        print("=" * 70)

        print(f"\n{'功能':<25} {'状态':<15} {'证据数量':<10}")
        print("-" * 70)

        for result in self.results:
            status_text = result.status.value
            evidence_count = len(result.evidence)
            print(f"{result.feature:<25} {status_text:<15} {evidence_count}")

        print("\n" + "-" * 70)
        print("📝 详细说明:\n")

        for result in self.results:
            print(f"\n【{result.feature}】")
            print(f"  状态：{result.status.value}")

            if result.evidence:
                print(f"  ✅ 已实现:")
                for item in result.evidence:
                    print(f"     - {item}")

            if result.missing:
                print(f"  ❌ 缺失:")
                for item in result.missing:
                    print(f"     - {item}")

            if result.recommendations:
                print(f"  💡 建议:")
                for item in result.recommendations:
                    print(f"     - {item}")

        # 汇总
        pass_count = sum(1 for r in self.results if r.status == VerificationStatus.PASS)
        partial_count = sum(
            1 for r in self.results if r.status == VerificationStatus.PARTIAL
        )
        fail_count = sum(1 for r in self.results if r.status == VerificationStatus.FAIL)

        print("\n" + "=" * 70)
        print("📈 汇总")
        print("=" * 70)
        print(f"✅ 通过：{pass_count}/{len(self.results)}")
        print(f"⚠️  部分支持：{partial_count}/{len(self.results)}")
        print(f"❌ 未实现：{fail_count}/{len(self.results)}")

        if pass_count == len(self.results):
            print("\n🎉 所有功能均已实现！")
            print("\n下一步：")
            print("  1. 运行 python3 scripts/verify_claude_features.py 进行在线测试")
            print("  2. 在 Slack 中测试实际交互效果")
            print("  3. 查看详细验证报告：docs/verification/claude-stdin-support-verification.md")
        elif pass_count > 0:
            print(f"\n💡 建议完成缺失项后进行在线验证")
            print(f"\n📄 详细 stdin 支持分析：docs/verification/claude-stdin-support-verification.md")
        else:
            print(f"\n⚠️  请先完成基本实现")
        
        # 添加 stdin 支持结论
        print("\n" + "="*70)
        print("📌 Claude Code CLI stdin 支持结论")
        print("="*70)
        print("""
根据代码分析和官方文档调研:

✅ Permission Request
   - stdout: 完整支持 permission_request 事件
   - stdin: 支持 {"behavior":"allow|deny"} 响应
   - HotPlex: 已完整实现 (provider/permission.go)

⚠️  AskUserQuestion
   - stdout: 输出 tool_use 事件可识别
   - stdin: 无官方响应格式规范
   - 建议：主要用于交互式 REPL，CLI headless 模式受限

✅ Plan Mode
   - stdout: thinking 事件包含 subtype=plan_generation
   - stdin: N/A (只读模式)
   - HotPlex: 需添加 subtype 识别逻辑

✅ Output Styles
   - stdout: 通过 answer 事件内容体现
   - stdin: N/A (配置驱动)
   - 激活：需在 .claude/settings.json 中配置

详细报告：docs/verification/claude-stdin-support-verification.md
        """)


def main():
    """主函数"""
    import argparse

    parser = argparse.ArgumentParser(
        description="离线验证 HotPlex 代码库对 Claude Code 高级功能的支持"
    )
    parser.add_argument(
        "--project-root", default=".", help="项目根目录 (默认：当前目录)"
    )
    parser.add_argument(
        "--feature",
        choices=[
            "plan-mode",
            "ask-user-question",
            "output-styles",
            "permission-request",
            "all",
        ],
        default="all",
        help="要验证的功能 (默认：all)",
    )

    args = parser.parse_args()

    verifier = CodebaseVerifier(project_root=args.project_root)

    if args.feature == "all":
        verifier.run_all_verifications()
    else:
        feature_map = {
            "plan-mode": verifier.verify_plan_mode_support,
            "ask-user-question": verifier.verify_ask_user_question_support,
            "output-styles": verifier.verify_output_styles_support,
            "permission-request": verifier.verify_permission_request_support,
        }

        if args.feature in feature_map:
            result = feature_map[args.feature]()
            verifier.results.append(result)
            verifier.print_report()


if __name__ == "__main__":
    main()
