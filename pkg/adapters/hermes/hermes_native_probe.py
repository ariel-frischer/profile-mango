from __future__ import annotations

import builtins
import hashlib
import importlib
import json
import os
from pathlib import Path
import socket
import sqlite3
import subprocess
import sys
import threading

EXPECTED_CONFIG_MODULE_SHA256 = "d76471ce54d40e68165e2cce7c2ade9c2164ed5ce4dbcf673b1b289cb89c7d84"
EXPECTED_VERSION_MODULE_SHA256 = "0d78a58a9f27f32adfdac959e89767cecde93a424bcfbd39d64fb95f6cf13e6c"
EXPECTED_SOURCE_COMMIT = "345cd2b057a452236de401d3534b8502a7465e8d"
EXPECTED_SOURCE_TREE = "6e14b9791cdc5a47068685e9429dd5d6bdc5ef5f"
EXPECTED_ARCHIVE_SHA256 = "71f2db39a64fbba282e3bd3be4b0f7b935585948a59a368d61deeec0f0827c47"
EXPECTED_CONFIG_PATH = "/mnt/home/.hermes/config.yaml"

state = Path(os.environ["HERMES_NATIVE_STATE"])
home = Path(os.environ["HERMES_HOME"])
result_path = state / "result" / "hermes-native-result.json"
source_root = Path(os.environ["HERMES_NATIVE_SOURCE"])


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def inventory(path: Path) -> list[dict[str, object]]:
    entries: list[dict[str, object]] = []
    if not path.exists():
        return entries
    for candidate in sorted(path.rglob("*")):
        relative = candidate.relative_to(path).as_posix()
        if candidate.is_symlink():
            entries.append({"path": relative, "kind": "symlink", "target": os.readlink(candidate)})
        elif candidate.is_dir():
            entries.append({"path": relative, "kind": "dir"})
        else:
            entries.append({"path": relative, "kind": "file", "size": candidate.stat().st_size})
    return entries


def forbidden(*_args: object, **_kwargs: object) -> None:
    raise RuntimeError("Hermes native config probe forbids external process, network, IPC, and database effects")


def install_effect_traps() -> None:
    socket.socket = forbidden  # type: ignore[assignment]
    for name in ("create_connection", "getaddrinfo", "gethostbyname", "gethostbyname_ex"):
        setattr(socket, name, forbidden)
    for name in ("Popen", "run", "call", "check_call", "check_output"):
        setattr(subprocess, name, forbidden)
    for name in ("system", "popen", "fork", "forkpty", "posix_spawn", "posix_spawnp"):
        if hasattr(os, name):
            setattr(os, name, forbidden)
    for name in dir(os):
        if name.startswith(("exec", "spawn")):
            setattr(os, name, forbidden)
    threading.Thread.start = forbidden  # type: ignore[method-assign]
    sqlite3.connect = forbidden  # type: ignore[assignment]


def audit_external_effect(event: str, _args: tuple[object, ...]) -> None:
    blocked = ("socket.", "subprocess.", "os.system", "os.exec", "os.spawn", "os.fork")
    if event.startswith(blocked):
        raise RuntimeError(f"audit blocked external effect: {event}")


def block_runtime_imports() -> None:
    blocked_prefixes = (
        "acp_adapter", "agent.session", "agent.transports", "agent.tools", "mcp",
        "hermes_cli.auth", "hermes_cli.daemon", "hermes_cli.hooks", "hermes_cli.plugins",
    )
    real_import = builtins.__import__

    def guarded_import(name: str, *args: object, **kwargs: object):
        if name == blocked_prefixes[0] or name.startswith(blocked_prefixes):
            raise ImportError(f"blocked native-probe import: {name}")
        return real_import(name, *args, **kwargs)

    builtins.__import__ = guarded_import


def nested(config: object, *keys: str) -> object:
    current = config
    for key in keys:
        if not isinstance(current, dict):
            return None
        current = current.get(key)
    return current


def write_result(payload: dict[str, object]) -> None:
    result_path.parent.mkdir(parents=True, exist_ok=True)
    result_path.write_text(json.dumps(payload, sort_keys=True, indent=2) + "\n", encoding="utf-8")


before = inventory(home)
try:
    sys.addaudithook(audit_external_effect)
    install_effect_traps()
    block_runtime_imports()
    sys.path.insert(0, str(source_root))
    site_packages = os.environ.get("PYTHONPATH", "").split(":")[-1]
    if site_packages:
        sys.path.insert(1, site_packages)

    if os.environ.get("HERMES_NATIVE_ARCHIVE_SHA256") != EXPECTED_ARCHIVE_SHA256:
        raise RuntimeError("source archive hash was not pinned by the runner")
    if os.environ.get("HERMES_NATIVE_SOURCE_COMMIT") != EXPECTED_SOURCE_COMMIT:
        raise RuntimeError("source commit was not pinned by the runner")
    if os.environ.get("HERMES_NATIVE_SOURCE_TREE") != EXPECTED_SOURCE_TREE:
        raise RuntimeError("source tree was not pinned by the runner")
    if os.environ.get("HERMES_NATIVE_CONFIG") != EXPECTED_CONFIG_PATH:
        raise RuntimeError("native probe config path was not pinned")
    if sha256_file(source_root / "hermes_cli" / "config.py") != EXPECTED_CONFIG_MODULE_SHA256:
        raise RuntimeError("config module hash changed before import")
    if sha256_file(source_root / "hermes_cli" / "__init__.py") != EXPECTED_VERSION_MODULE_SHA256:
        raise RuntimeError("version module hash changed before import")

    config_module = importlib.import_module("hermes_cli.config")
    version_module = importlib.import_module("hermes_cli")
    config_file = Path(config_module.__file__).resolve()
    version_file = Path(version_module.__file__).resolve()
    if config_file != source_root / "hermes_cli" / "config.py":
        raise RuntimeError(f"config module loaded from unexpected path: {config_file}")
    if version_file != source_root / "hermes_cli" / "__init__.py":
        raise RuntimeError(f"version module loaded from unexpected path: {version_file}")
    imported_config_hash = sha256_file(config_file)
    imported_version_hash = sha256_file(version_file)
    if imported_config_hash != EXPECTED_CONFIG_MODULE_SHA256:
        raise RuntimeError("imported config module hash changed")
    if imported_version_hash != EXPECTED_VERSION_MODULE_SHA256:
        raise RuntimeError("imported version module hash changed")

    loaded = config_module.load_config_readonly()
    selected = {
        "model": {
            "provider": nested(loaded, "model", "provider"),
            "default": nested(loaded, "model", "default"),
        },
        "agent": {"reasoning_effort": nested(loaded, "agent", "reasoning_effort")},
    }
    if selected["model"]["provider"] != "openai":
        raise RuntimeError(f"native provider merge mismatch: {selected}")
    if selected["model"]["default"] != "gpt-5.6":
        raise RuntimeError(f"native model merge mismatch: {selected}")
    if selected["agent"]["reasoning_effort"] != "high":
        raise RuntimeError(f"native reasoning merge mismatch: {selected}")
    unknown_sentinel = nested(loaded, "unknown_sentinel")
    if nested(loaded, "unknown_sentinel", "value") != "SYNTHETIC_SECRET_SENTINEL":
        raise RuntimeError("native effective merge dropped the unrelated sentinel")

    hidden_real_paths = all(
        not any(path.iterdir()) for path in (Path("/home"), Path("/root"), Path("/run"))
    )
    pid_namespace_isolated = os.getppid() == 1 and os.getpid() != 1
    if not hidden_real_paths:
        raise RuntimeError("real home or socket paths are visible in the probe")
    if not pid_namespace_isolated:
        raise RuntimeError(f"PID namespace is not isolated, pid={os.getpid()}")
    expected_env = {
        "HOME", "HERMES_HOME", "HERMES_MANAGED_DIR", "HERMES_NATIVE_STATE", "HERMES_NATIVE_SOURCE",
        "HERMES_NATIVE_CONFIG", "HERMES_NATIVE_ARCHIVE_SHA256", "HERMES_NATIVE_SOURCE_COMMIT",
        "HERMES_NATIVE_SOURCE_TREE", "HERMES_NATIVE_CONFIG_MODULE_SHA256",
        "HERMES_NATIVE_VERSION_MODULE_SHA256", "HERMES_NATIVE_ISOLATION_FLAGS", "PATH", "PYTHONHOME",
        "PYTHONPATH", "PYTHONNOUSERSITE", "PYTHONDONTWRITEBYTECODE", "XDG_CONFIG_HOME",
        "XDG_CACHE_HOME", "XDG_DATA_HOME", "TMPDIR", "LANG", "LC_ALL", "TERM", "NO_COLOR", "PWD",
    }
    unexpected_env = sorted(set(os.environ) - expected_env)
    if unexpected_env:
        raise RuntimeError(f"clearenv leaked unexpected environment names: {unexpected_env}")

    output = {
        "status": "ok",
        "target": "Hermes Agent 0.21.3",
        "source_release": "v2026.9.14",
        "source_commit": os.environ["HERMES_NATIVE_SOURCE_COMMIT"],
        "source_tree": os.environ["HERMES_NATIVE_SOURCE_TREE"],
        "source_archive_sha256": os.environ["HERMES_NATIVE_ARCHIVE_SHA256"],
        "native_loader": "hermes_cli.config.load_config_readonly",
        "config_module": "hermes_cli/config.py",
        "config_module_sha256": imported_config_hash,
        "version_module": "hermes_cli/__init__.py",
        "version_module_sha256": imported_version_hash,
        "config_path": str(Path(os.environ["HERMES_NATIVE_CONFIG"])),
        "selected": selected,
        "unknown_sentinel": unknown_sentinel,
        "before": before,
        "after": inventory(home),
        "clearenv": True,
        "hidden_real_home_root_run_paths": hidden_real_paths,
        "pid_namespace_isolated": pid_namespace_isolated,
        "pid": os.getpid(),
        "parent_pid": os.getppid(),
        "isolation_flags": os.environ["HERMES_NATIVE_ISOLATION_FLAGS"].split(),
        "effect_traps": {
            "network": "socket APIs and audit events trapped",
            "subprocess": "subprocess APIs and audit events trapped",
            "ipc": "socket and sqlite APIs trapped",
            "threads": "thread start trapped",
        },
        "runtime_scope": "config module import and effective merge only",
    }
except Exception as exc:
    write_result({
        "status": "error",
        "error": f"{type(exc).__name__}: {exc}",
        "before": before,
        "after": inventory(home),
    })
    raise
else:
    write_result(output)
