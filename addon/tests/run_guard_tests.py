"""Run guard regressions in isolated native Lua 5.1 runtimes (pip install lupa)."""
import argparse
import sys
from pathlib import Path

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument("--runtime-path", help="Optional directory containing an isolated lupa installation")
parser.add_argument("--reference", help="Installed TRP3 ScriptGeneration.lua for the real-path smoke")
parser.add_argument("files", nargs="*", help="Specific Lua tests; defaults to every item_guard smoke")
options = parser.parse_args()
if options.runtime_path:
    sys.path.insert(0, options.runtime_path)
from lupa.lua51 import LuaRuntime

files = [Path(name) for name in options.files] or sorted(Path("addon/tests").glob("item_guard*_smoke.lua"))
for file in files:
    runtime = LuaRuntime()
    if options.reference:
        runtime.globals().arg = runtime.table_from({1: options.reference})
    runtime.execute("local path = ...; assert(loadfile(path))()", str(file).replace("\\", "/"))
    print(f"PASS Lua 5.1: {file}", flush=True)
