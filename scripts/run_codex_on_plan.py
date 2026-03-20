#!/usr/bin/env python3
"""Run Codex against a plan file in multiple rounds until all tasks are done."""

from __future__ import annotations

import argparse
import json
import logging
import shlex
import shutil
import subprocess
import sys
import time
from pathlib import Path
from typing import Optional

DONE_TEXT = "计划已经全部完成"
NOT_DONE_TEXT = "计划还没有全部完成"


def setup_logger(log_file: Optional[Path], verbose: bool) -> logging.Logger:
    logger = logging.getLogger("run_codex_on_plan")
    logger.handlers.clear()
    logger.setLevel(logging.DEBUG if verbose else logging.INFO)
    formatter = logging.Formatter("%(asctime)s | %(levelname)s | %(message)s")

    console = logging.StreamHandler(sys.stderr)
    console.setFormatter(formatter)
    console.setLevel(logging.DEBUG if verbose else logging.INFO)
    logger.addHandler(console)

    if log_file is not None:
        log_file.parent.mkdir(parents=True, exist_ok=True)
        file_handler = logging.FileHandler(log_file, encoding="utf-8")
        file_handler.setFormatter(formatter)
        file_handler.setLevel(logging.DEBUG)
        logger.addHandler(file_handler)

    return logger


def default_context_file(plan_file: Path) -> Path:
    return plan_file.with_suffix(".json")


def default_log_file(plan_file: Path) -> Path:
    return plan_file.with_suffix(".log")


def build_exec_base_cmd(codex_bin: str, sandbox: bool) -> list[str]:
    cmd = [codex_bin, "exec", "--skip-git-repo-check", "--json"]
    if sandbox:
        cmd.append("--full-auto")
    else:
        cmd.append("--dangerously-bypass-approvals-and-sandbox")
    return cmd


def build_resume_cmd(codex_bin: str, thread_id: str, sandbox: bool) -> list[str]:
    cmd = [codex_bin, "exec", "resume", thread_id, "--skip-git-repo-check", "--json"]
    if sandbox:
        cmd.append("--full-auto")
    else:
        cmd.append("--dangerously-bypass-approvals-and-sandbox")
    return cmd


def run_codex(command: list[str], prompt: str, timeout_sec: int) -> tuple[int, str, str]:
    full_cmd = command + [prompt]
    try:
        proc = subprocess.run(
            full_cmd,
            text=True,
            capture_output=True,
            timeout=timeout_sec,
            check=False,
        )
    except subprocess.TimeoutExpired as exc:
        out = exc.stdout or ""
        err = exc.stderr or ""
        return 124, out, f"调用 codex 超时（{timeout_sec} 秒）\n{err}"
    except OSError as exc:
        return 1, "", str(exc)
    return proc.returncode, proc.stdout, proc.stderr


def run_codex_with_retry(
    command: list[str],
    prompt: str,
    timeout_sec: int,
    max_retry: int,
    retry_wait_sec: int,
    logger: logging.Logger,
    stage: str,
) -> tuple[int, str, str]:
    rc = 1
    out = ""
    err = ""
    for attempt in range(max_retry + 1):
        rc, out, err = run_codex(command, prompt, timeout_sec)
        if rc == 0:
            return rc, out, err
        logger.error("%s 失败，退出码=%s", stage, rc)
        if err.strip():
            logger.error("%s 标准错误:\n%s", stage, err.strip())
        if attempt < max_retry:
            logger.info("将在 %s 秒后重试（%s/%s）", retry_wait_sec, attempt + 1, max_retry)
            time.sleep(retry_wait_sec)
    return rc, out, err


def parse_jsonl_for_thread_and_last_message(output: str) -> tuple[Optional[str], Optional[str]]:
    thread_id: Optional[str] = None
    last_message: Optional[str] = None
    for raw_line in output.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            continue
        if not isinstance(event, dict):
            continue
        if event.get("type") == "thread.started":
            candidate = event.get("thread_id")
            if isinstance(candidate, str) and candidate.strip():
                thread_id = candidate.strip()
        if event.get("type") == "item.completed":
            item = event.get("item")
            if isinstance(item, dict) and item.get("type") == "agent_message":
                text = item.get("text")
                if isinstance(text, str) and text.strip():
                    last_message = text.strip()
    return thread_id, last_message


def normalize_check_answer(last_message: Optional[str], raw_output: str) -> str:
    text = (last_message or "").strip()
    if not text:
        text = raw_output.strip()
    if DONE_TEXT in text:
        return DONE_TEXT
    if NOT_DONE_TEXT in text:
        return NOT_DONE_TEXT
    return NOT_DONE_TEXT


def load_context(path: Path) -> Optional[str]:
    if not path.exists():
        return None
    try:
        payload = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return None
    thread_id = payload.get("thread_id")
    if isinstance(thread_id, str) and thread_id.strip():
        return thread_id.strip()
    return None


def save_context(path: Path, thread_id: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    payload = {"thread_id": thread_id}
    path.write_text(json.dumps(payload, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def ensure_status_file(path: Path) -> None:
    if path.exists():
        return
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text("# 计划状态\n\n", encoding="utf-8")


def format_cmd(command: list[str], prompt: str) -> str:
    return shlex.join(command + [prompt])


def build_work_prompt(plan_file: Path, status_file: Optional[Path], iteration: int, max_iteration: int) -> str:
    status_line = f"- 状态文件: {status_file}\n" if status_file else "- 状态文件: 未提供（可不写）\n"
    return (
        "你是开发执行代理，请按计划推进工作。\n"
        f"- 计划文件: {plan_file}\n"
        f"{status_line}"
        f"- 当前轮次: {iteration}/{max_iteration}\n"
        "要求：\n"
        "1) 读取计划后可连续完成多个任务，不要只做一步。\n"
        "2) 完成开发后执行必要的 lint/测试验证。\n"
        "3) 如果提供了状态文件，可记录本轮进度；未提供则跳过。\n"
        "4) 本轮结束时总结已完成任务、未完成任务、关键验证命令及结果。\n"
    )


def build_check_prompt(plan_file: Path, status_file: Optional[Path]) -> str:
    status_line = f"状态文件（可选参考）: {status_file}\n" if status_file else "状态文件: 未提供\n"
    return (
        "请严格检查计划完成度。\n"
        f"计划文件: {plan_file}\n"
        f"{status_line}"
        "要求：逐项检查计划任务与当前仓库状态，不得只依赖历史上下文。\n"
        f"只能回答 {DONE_TEXT} 或 {NOT_DONE_TEXT}，不能输出其他字符。"
    )


def resolve_path(path: Path) -> Path:
    return path.expanduser().resolve()


def execute(args: argparse.Namespace, logger: logging.Logger) -> int:
    plan_file = resolve_path(Path(args.plan_file))
    if not plan_file.exists():
        logger.error("计划文件不存在: %s", plan_file)
        return 2

    status_file = resolve_path(Path(args.status_file)) if args.status_file else None
    context_file = resolve_path(Path(args.context_file)) if args.context_file else default_context_file(plan_file)
    log_file = resolve_path(Path(args.log_file)) if args.log_file else default_log_file(plan_file)

    if status_file is not None:
        ensure_status_file(status_file)

    if args.session_id and not args.resume_context:
        logger.warning("传入 --session-id 时会自动启用 --resume-context")
        args.resume_context = True

    if not args.dry_run and not shutil.which(args.codex_bin):
        logger.error("找不到 codex 可执行文件: %s", args.codex_bin)
        return 2

    exec_base_cmd = build_exec_base_cmd(args.codex_bin, args.sandbox)
    thread_id = args.session_id
    if args.resume_context and thread_id is None:
        thread_id = load_context(context_file)

    logger.info(
        "开始执行: plan=%s status=%s context=%s sandbox=%s resume=%s dry_run=%s max_iteration=%s",
        plan_file,
        status_file if status_file else "无",
        context_file,
        args.sandbox,
        args.resume_context,
        args.dry_run,
        args.max_iteration,
    )

    for idx in range(1, args.max_iteration + 1):
        work_prompt = build_work_prompt(plan_file, status_file, idx, args.max_iteration)
        work_cmd = (
            build_resume_cmd(args.codex_bin, thread_id, args.sandbox)
            if args.resume_context and thread_id
            else exec_base_cmd
        )

        logger.info("第 %s 轮: 工作执行", idx)
        if args.dry_run:
            logger.info("[dry-run] 命令: %s", format_cmd(work_cmd, work_prompt))
        else:
            rc, out, _ = run_codex_with_retry(
                work_cmd,
                work_prompt,
                args.timeout_sec,
                args.max_retry,
                args.retry_wait_sec,
                logger,
                "工作轮次",
            )
            if rc != 0:
                return rc
            new_thread_id, _ = parse_jsonl_for_thread_and_last_message(out)
            if new_thread_id:
                thread_id = new_thread_id
                if args.resume_context:
                    save_context(context_file, thread_id)

        check_prompt = build_check_prompt(plan_file, status_file)
        check_cmd = (
            build_resume_cmd(args.codex_bin, thread_id, args.sandbox)
            if args.resume_context and thread_id
            else exec_base_cmd
        )
        logger.info("第 %s 轮: 完成度检查", idx)

        if args.dry_run:
            logger.info("[dry-run] 命令: %s", format_cmd(check_cmd, check_prompt))
            continue

        rc, out, _ = run_codex_with_retry(
            check_cmd,
            check_prompt,
            args.timeout_sec,
            args.max_retry,
            args.retry_wait_sec,
            logger,
            "完成度检查",
        )
        if rc != 0:
            return rc

        new_thread_id, last_message = parse_jsonl_for_thread_and_last_message(out)
        if new_thread_id:
            thread_id = new_thread_id
            if args.resume_context:
                save_context(context_file, thread_id)

        answer = normalize_check_answer(last_message, out)
        logger.info("完成度回答: %s", answer)
        if answer == DONE_TEXT:
            logger.info("计划在第 %s 轮完成", idx)
            return 0

    if args.dry_run:
        logger.info("dry-run 完成（未实际调用 codex）")
        return 0

    logger.error("达到最大轮次后计划仍未完成: %s", args.max_iteration)
    logger.error("可查看日志: %s", log_file)
    return 1


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="以非交互方式按计划驱动 codex 多轮执行，直到任务完成。",
    )
    parser.add_argument("-P", "--plan-file", required=True, help="计划文件路径")
    parser.add_argument("-S", "--status-file", help="状态文件路径（可选）")
    parser.add_argument("-C", "--context-file", help="线程上下文文件（默认: <plan>.json）")
    parser.add_argument("-L", "--log-file", help="日志文件（默认: <plan>.log）")
    parser.add_argument("-M", "--max-iteration", type=int, default=200, help="最大轮次")
    parser.add_argument("--sandbox", dest="sandbox", action="store_true", default=True, help="启用沙盒")
    parser.add_argument("--no-sandbox", dest="sandbox", action="store_false", help="关闭沙盒")
    parser.add_argument(
        "--resume-context",
        dest="resume_context",
        action="store_true",
        default=False,
        help="启用多轮上下文复用（codex resume）",
    )
    parser.add_argument("--no-resume-context", dest="resume_context", action="store_false", help="关闭上下文复用")
    parser.add_argument("-s", "--session-id", help="指定已有 codex thread id")
    parser.add_argument("--dry-run", action="store_true", help="仅打印命令，不执行")
    parser.add_argument("--timeout-sec", type=int, default=7200, help="单轮超时（秒）")
    parser.add_argument("--codex-bin", default="codex", help="codex 可执行文件")
    parser.add_argument("--max-retry", type=int, default=10, help="失败重试次数")
    parser.add_argument("--retry-wait-sec", type=int, default=60, help="重试等待秒数")
    parser.add_argument("--verbose", action="store_true", help="输出调试日志")
    args = parser.parse_args()

    if args.max_iteration < 1:
        parser.error("--max-iteration 必须 >= 1")
    if args.timeout_sec < 1:
        parser.error("--timeout-sec 必须 >= 1")
    if args.max_retry < 0:
        parser.error("--max-retry 必须 >= 0")
    if args.retry_wait_sec < 0:
        parser.error("--retry-wait-sec 必须 >= 0")
    return args


def main() -> int:
    args = parse_args()
    plan_file = resolve_path(Path(args.plan_file))
    log_file = resolve_path(Path(args.log_file)) if args.log_file else default_log_file(plan_file)
    logger = setup_logger(log_file, args.verbose)
    return execute(args, logger)


if __name__ == "__main__":
    raise SystemExit(main())
