#!/usr/bin/env python3
"""
Fetches the PoE OpenAPI spec, marks required properties with
x-oapi-codegen-extra-tags so oapi-codegen emits `binding:"required"`
on the corresponding Go fields (needed for our own swagger generation,
which derives "required" purely from binding/validate struct tags),
and regenerates client/generated-types.go.
"""
import json
import subprocess
import sys
import tempfile
import urllib.request
from pathlib import Path

SPEC_URL = "https://liberatorist.github.io/poe-openapi/out/openapi-poe1.json"
REPO_ROOT = Path(__file__).resolve().parent.parent
OUTPUT_FILE = REPO_ROOT / "client" / "generated-types.go"


def add_required_tags(spec: dict) -> None:
    for schema in spec.get("components", {}).get("schemas", {}).values():
        required = schema.get("required", [])
        properties = schema.get("properties", {})
        for name in required:
            prop = properties.get(name)
            if prop is None:
                continue
            prop.setdefault("x-oapi-codegen-extra-tags", {})["binding"] = "required"


def main() -> None:
    with urllib.request.urlopen(SPEC_URL) as resp:
        spec = json.load(resp)

    add_required_tags(spec)

    with tempfile.TemporaryDirectory() as tmp:
        spec_path = Path(tmp) / "openapi-poe1.json"
        spec_path.write_text(json.dumps(spec))

        config_path = Path(tmp) / "config.yaml"
        config_path.write_text(
            "package: client\n"
            "generate:\n"
            "  models: true\n"
        )

        subprocess.run(
            [
                "oapi-codegen",
                "-config",
                str(config_path),
                "-o",
                str(OUTPUT_FILE),
                str(spec_path),
            ],
            check=True,
        )


if __name__ == "__main__":
    sys.exit(main())
