"""Script to generate Python gRPC stubs from proto file."""

import subprocess
import sys
from pathlib import Path


def generate():
    proto_dir = Path(__file__).parent.parent / "protos"
    out_dir = Path(__file__).parent / "adapter" / "grpc" / "generated"
    out_dir.mkdir(parents=True, exist_ok=True)

    # Create __init__.py
    (out_dir / "__init__.py").touch()

    cmd = [
        sys.executable,
        "-m",
        "grpc_tools.protoc",
        f"-I{proto_dir}",
        f"--python_out={out_dir}",
        f"--grpc_python_out={out_dir}",
        f"--pyi_out={out_dir}",
        str(proto_dir / "ai_service.proto"),
    ]

    print(f"Running: {' '.join(cmd)}")
    subprocess.run(cmd, check=True)
    print(f"Generated stubs in {out_dir}")

    # Fix imports in generated grpc file
    grpc_file = out_dir / "ai_service_pb2_grpc.py"
    if grpc_file.exists():
        content = grpc_file.read_text()
        content = content.replace(
            "import ai_service_pb2",
            "from . import ai_service_pb2",
        )
        grpc_file.write_text(content)
        print("Fixed imports in ai_service_pb2_grpc.py")


if __name__ == "__main__":
    generate()
