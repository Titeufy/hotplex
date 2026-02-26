#!/usr/bin/env python3
"""
Claude Code Advanced Features Verification Script

验证 Claude Code CLI 对以下高级功能的支持：
1. Plan Mode - 只生成计划不执行
2. AskUserQuestion 工具 - 澄清问题的交互工具
3. Output Styles (Learning/Explanatory) - 教育性输出风格

测试方法：
- 使用 claude -p 命令发送测试提示词
- 捕获 stream-json 输出
- 解析并验证事件类型

使用前确保：
1. 已安装 Claude Code CLI: npm install -g @anthropic-ai/claude-code
2. 已完成认证：claude auth login
3. Node.js 版本 >= 18

运行方式：
    python3 scripts/verify_claude_features.py

作者：HotPlex Team
日期：2026-02-26
"""

import subprocess
import json
import sys
import os
from typing import Any, Optional
from dataclasses import dataclass, asdict
from enum import Enum


class FeatureStatus(Enum):
    """功能支持状态"""

    SUPPORTED = "✅ 支持"
    PARTIAL = "⚠️ 部分支持"
    NOT_SUPPORTED = "❌ 不支持"
    ERROR = "❌ 错误"


@dataclass
class TestResult:
    """测试结果"""

    feature: str
    status: FeatureStatus
    events_found: list[str]
    evidence: str
    notes: str = ""


@dataclass
class ClaudeEvent:
    """Claude Code 事件"""

    type: str
    subtype: Optional[str] = None
    name: Optional[str] = None
    content: Optional[str] = None
    raw: dict = None


class ClaudeCodeVerifier:
    """Claude Code 功能验证器"""

    def __init__(self, work_dir: str = "."):
        self.work_dir = work_dir
        self.claude_version: Optional[str] = None
        self.test_results: list[TestResult] = []

    def check_claude_installed(self) -> bool:
        """检查 Claude Code CLI 是否已安装"""
        print("\n📦 检查 Claude Code CLI 安装...")
        try:
            result = subprocess.run(
                ["claude", "--version"], capture_output=True, text=True, timeout=10
            )
            if result.returncode == 0:
                self.claude_version = result.stdout.strip()
                print(f"✅ 已安装：{self.claude_version}")
                return True
            else:
                print(f"❌ 检查失败：{result.stderr}")
                return False
        except FileNotFoundError:
            print(
                "❌ 未找到 claude 命令，请先安装：npm install -g @anthropic-ai/claude-code"
            )
            return False
        except subprocess.TimeoutExpired:
            print("❌ 检查超时")
            return False

    def check_auth_status(self) -> bool:
        """检查认证状态"""
        print("\n🔐 检查认证状态...")
        try:
            result = subprocess.run(
                ["claude", "auth", "status", "--text"],
                capture_output=True,
                text=True,
                timeout=10,
            )
            if result.returncode == 0:
                print(f"✅ 已认证")
                return True
            else:
                print(f"❌ 未认证，请先运行：claude auth login")
                return False
        except Exception as e:
            print(f"❌ 检查失败：{e}")
            return False

    def run_claude_command(
        self, prompt: str, extra_args: list[str] = None
    ) -> tuple[bool, str, list[ClaudeEvent]]:
        """
        运行 Claude Code 命令并解析输出

        Returns:
            (success, raw_output, events)
        """
        args = [
            "claude",
            "-p",
            prompt,
            "--output-format",
            "stream-json",
            "--verbose",
            "--include-partial-messages",
        ]

        if extra_args:
            args.extend(extra_args)

        print(f"\n▶️  执行：{' '.join(args)}")

        try:
            result = subprocess.run(
                args, capture_output=True, text=True, timeout=60, cwd=self.work_dir
            )

            events = []
            raw_output = result.stdout

            # 解析 JSON Lines
            for line in raw_output.strip().split("\n"):
                if not line.strip():
                    continue
                try:
                    event_data = json.loads(line)
                    event = ClaudeEvent(
                        type=event_data.get("type", "unknown"),
                        subtype=event_data.get("subtype"),
                        name=event_data.get("name"),
                        content=event_data.get("content", "")
                        or event_data.get("output", ""),
                        raw=event_data,
                    )
                    events.append(event)
                except json.JSONDecodeError:
                    # 非 JSON 行，跳过
                    pass

            return result.returncode == 0, raw_output, events

        except subprocess.TimeoutExpired:
            return False, "命令超时", []
        except Exception as e:
            return False, str(e), []

    def verify_plan_mode(self) -> TestResult:
        """
        验证 Plan Mode 支持

        Plan Mode 应该只生成计划，不执行工具调用
        """
        print("\n" + "=" * 60)
        print("测试 1: Plan Mode (计划模式)")
        print("=" * 60)

        prompt = """
        分析这个项目的结构，制定一个重构计划。
        只需要输出计划步骤，不要执行任何操作。
        """

        # Plan Mode 通常通过 /plan 命令或交互式切换
        # 在 CLI 中，我们通过提示词来测试是否会产生计划
        success, raw_output, events = self.run_claude_command(prompt)

        if not success:
            return TestResult(
                feature="Plan Mode",
                status=FeatureStatus.ERROR,
                events_found=[],
                evidence=raw_output,
                notes="命令执行失败",
            )

        # 分析事件
        event_types = [e.type for e in events]
        has_tool_use = "tool_use" in event_types
        has_thinking = "thinking" in event_types
        has_plan_subtype = any(
            e.subtype == "plan_generation" for e in events if e.type == "thinking"
        )

        # Plan Mode 特征：有 thinking 事件，subtype=plan_generation，无 tool_use
        if has_plan_subtype:
            status = FeatureStatus.SUPPORTED
            notes = "检测到 subtype=plan_generation 事件"
        elif has_thinking and not has_tool_use:
            status = FeatureStatus.PARTIAL
            notes = "有思考过程但无工具调用（可能是提示词效果）"
        else:
            status = FeatureStatus.PARTIAL
            notes = "未检测到明确的 Plan Mode 标记，需要交互式切换"

        # 提取证据
        plan_events = [
            e for e in events if e.type == "thinking" and e.subtype == "plan_generation"
        ]
        evidence = (
            f"发现 {len(plan_events)} 个 plan_generation 事件"
            if plan_events
            else "无 plan_generation 事件"
        )

        return TestResult(
            feature="Plan Mode",
            status=status,
            events_found=list(set(event_types)),
            evidence=evidence,
            notes=notes,
        )

    def verify_ask_user_question(self) -> TestResult:
        """
        验证 AskUserQuestion 工具支持

        该工具在 v2.0.21+ 引入，用于澄清问题
        """
        print("\n" + "=" * 60)
        print("测试 2: AskUserQuestion 工具")
        print("=" * 60)

        # 创建一个需要澄清的场景
        prompt = """
        我想添加一个新的功能，但是不确定应该使用什么技术栈。
        请问我一些问题来帮助我明确需求。
        如果有多个选项，请提供可选择的列表。
        """

        success, raw_output, events = self.run_claude_command(prompt)

        if not success:
            return TestResult(
                feature="AskUserQuestion",
                status=FeatureStatus.ERROR,
                events_found=[],
                evidence=raw_output,
                notes="命令执行失败",
            )

        # 分析事件
        event_types = [e.type for e in events]
        tool_names = [e.name for e in events if e.type == "tool_use"]

        has_ask_tool = "AskUserQuestion" in tool_names
        has_tool_use = "tool_use" in event_types

        # 检查是否有 AskUserQuestion 工具调用
        ask_events = [e for e in events if e.name == "AskUserQuestion"]

        if has_ask_tool:
            status = FeatureStatus.SUPPORTED
            notes = f"检测到 AskUserQuestion 工具调用 ({len(ask_events)} 次)"
        elif has_tool_use:
            status = FeatureStatus.PARTIAL
            notes = f"有其他工具调用，但未使用 AskUserQuestion: {set(tool_names)}"
        else:
            status = FeatureStatus.PARTIAL
            notes = "未检测到工具调用（可能是版本问题或 Claude 自主决策）"

        # 提取证据
        if ask_events:
            evidence = f"AskUserQuestion 调用详情:\n{json.dumps(ask_events[0].raw, indent=2, ensure_ascii=False)[:500]}..."
        else:
            evidence = f"工具调用列表：{set(tool_names)}"

        return TestResult(
            feature="AskUserQuestion",
            status=status,
            events_found=list(set(event_types)),
            evidence=evidence,
            notes=notes,
        )

    def verify_output_styles(self) -> TestResult:
        """
        验证 Output Styles 支持

        - Explanatory: 提供教育性见解
        - Learning: 协作学习模式，添加 TODO(human)
        """
        print("\n" + "=" * 60)
        print("测试 3: Output Styles (输出风格)")
        print("=" * 60)

        # 测试 Explanatory Style
        prompt_explanatory = """
        请解释这段代码的工作原理，并提供教育性的见解。
        使用 explanatory output style。
        """

        # 测试 Learning Style
        prompt_learning = """
        请教我如何编写一个 HTTP 服务器。
        使用 learning output style，在我理解后让我自己实现部分代码。
        """

        results = []

        # 测试 Explanatory
        print("\n  测试 Explanatory Style...")
        success_exp, raw_exp, events_exp = self.run_claude_command(prompt_explanatory)

        # 测试 Learning
        print("  测试 Learning Style...")
        success_learn, raw_learn, events_learn = self.run_claude_command(
            prompt_learning
        )

        # 分析结果
        has_explanatory = False
        has_learning = False
        has_todo_human = False

        # 检查是否检测到 output style 相关标记
        all_events = events_exp + events_learn

        # 检查 TODO(human) 标记 (Learning Style 特征)
        for event in events_learn:
            if event.content and "TODO(human)" in event.content:
                has_todo_human = True
                break

        # 检查是否有教育性见解标记
        for event in all_events:
            content_lower = (event.content or "").lower()
            if "insight" in content_lower or "explanation" in content_lower:
                has_explanatory = True
                break

        if has_todo_human:
            status = FeatureStatus.SUPPORTED
            notes = "Learning Style 检测到 TODO(human) 标记"
        elif has_explanatory:
            status = FeatureStatus.PARTIAL
            notes = "Explanatory Style 有教育性内容，但需要配置切换"
        else:
            status = FeatureStatus.PARTIAL
            notes = "Output Styles 需要通过 /output-style 命令切换，或配置文件设置"

        evidence = []
        if has_todo_human:
            evidence.append("✅ Learning: 检测到 TODO(human)")
        if has_explanatory:
            evidence.append("✅ Explanatory: 检测到教育性内容")
        if not evidence:
            evidence.append("⚠️ 需要在 .claude/settings.json 中配置 outputStyle 字段")

        return TestResult(
            feature="Output Styles",
            status=status,
            events_found=[e.type for e in all_events],
            evidence="\n".join(evidence),
            notes=notes,
        )

    def verify_permission_request(self) -> TestResult:
        """
        验证 Permission Request 支持

        在非 bypass-permissions 模式下，危险操作会触发权限请求
        """
        print("\n" + "=" * 60)
        print("测试 4: Permission Request (权限请求)")
        print("=" * 60)

        # 创建一个需要权限的操作
        prompt = """
        列出当前目录下的所有文件。
        """

        # 使用默认权限模式（非 bypass）
        success, raw_output, events = self.run_claude_command(
            prompt, extra_args=["--permission-mode", "default"]
        )

        if not success:
            return TestResult(
                feature="Permission Request",
                status=FeatureStatus.ERROR,
                events_found=[],
                evidence=raw_output,
                notes="命令执行失败",
            )

        # 分析事件
        event_types = [e.type for e in events]
        has_permission = "permission_request" in event_types

        if has_permission:
            status = FeatureStatus.SUPPORTED
            notes = "检测到 permission_request 事件"
        else:
            # 可能命令太简单不需要权限
            status = FeatureStatus.PARTIAL
            notes = "未检测到 permission_request (可能命令太简单或权限模式配置问题)"

        # 提取证据
        perm_events = [e for e in events if e.type == "permission_request"]
        if perm_events:
            evidence = f"Permission Request 详情:\n{json.dumps(perm_events[0].raw, indent=2, ensure_ascii=False)[:500]}..."
        else:
            evidence = f"事件类型：{set(event_types)}"

        return TestResult(
            feature="Permission Request",
            status=status,
            events_found=list(set(event_types)),
            evidence=evidence,
            notes=notes,
        )

    def run_all_tests(self):
        """运行所有测试"""
        print("\n" + "🚀" * 30)
        print("Claude Code 高级功能验证")
        print("🚀" * 30)

        # 前置检查
        if not self.check_claude_installed():
            print("\n❌ Claude Code CLI 未安装，无法继续测试")
            return

        if not self.check_auth_status():
            print("\n❌ 未认证，请先运行：claude auth login")
            return

        # 运行测试
        self.test_results.append(self.verify_plan_mode())
        self.test_results.append(self.verify_ask_user_question())
        self.test_results.append(self.verify_output_styles())
        self.test_results.append(self.verify_permission_request())

        # 输出报告
        self.print_report()

    def print_report(self):
        """打印测试报告"""
        print("\n" + "=" * 70)
        print("📊 测试报告")
        print("=" * 70)

        print(f"\n{'功能':<25} {'状态':<15} {'发现的事件类型'}")
        print("-" * 70)

        for result in self.test_results:
            status_emoji = result.status.value
            events_str = ", ".join(result.events_found[:5])  # 限制显示
            print(f"{result.feature:<25} {status_emoji:<15} {events_str}")

        print("\n" + "-" * 70)
        print("📝 详细说明:\n")

        for result in self.test_results:
            print(f"\n【{result.feature}】")
            print(f"  状态：{result.status.value}")
            print(f"  说明：{result.notes}")
            print(f"  证据：{result.evidence[:200]}...")

        # 总结
        supported_count = sum(
            1 for r in self.test_results if r.status == FeatureStatus.SUPPORTED
        )
        partial_count = sum(
            1 for r in self.test_results if r.status == FeatureStatus.PARTIAL
        )
        error_count = sum(
            1 for r in self.test_results if r.status == FeatureStatus.ERROR
        )

        print("\n" + "=" * 70)
        print("📈 汇总")
        print("=" * 70)
        print(f"✅ 完全支持：{supported_count}")
        print(f"⚠️  部分支持：{partial_count}")
        print(f"❌ 不支持/错误：{error_count}")

        if supported_count == len(self.test_results):
            print("\n🎉 所有功能均得到支持！")
        elif supported_count > 0:
            print(f"\n💡 建议：部分功能需要交互式环境或配置切换")
        else:
            print(f"\n⚠️  请检查 Claude Code 版本和网络连接")


def main():
    """主函数"""
    import argparse

    parser = argparse.ArgumentParser(
        description="验证 Claude Code CLI 对高级功能的支持"
    )
    parser.add_argument(
        "--work-dir", default=os.getcwd(), help="工作目录 (默认：当前目录)"
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
        help="要测试的功能 (默认：all)",
    )

    args = parser.parse_args()

    verifier = ClaudeCodeVerifier(work_dir=args.work_dir)

    if args.feature == "all":
        verifier.run_all_tests()
    else:
        # 单个功能测试
        if not verifier.check_claude_installed():
            return
        if not verifier.check_auth_status():
            return

        feature_map = {
            "plan-mode": verifier.verify_plan_mode,
            "ask-user-question": verifier.verify_ask_user_question,
            "output-styles": verifier.verify_output_styles,
            "permission-request": verifier.verify_permission_request,
        }

        result = feature_map[args.feature]()
        verifier.test_results.append(result)
        verifier.print_report()


if __name__ == "__main__":
    main()
