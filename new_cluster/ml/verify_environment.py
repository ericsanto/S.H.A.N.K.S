import argparse
import json
import sys

import numpy
import pyspark
import torch
from pyspark.ml.torch.distributor import TorchDistributor  # noqa: F401


parser = argparse.ArgumentParser()
parser.add_argument("--gpu", action="store_true")
parser.add_argument("--flavor", choices=["cpu", "cu126"], default="cpu")
args = parser.parse_args()

assert sys.version_info[:3] == (3, 11, 14), sys.version
assert pyspark.__version__ == "3.5.3", pyspark.__version__
assert torch.__version__.split("+")[0] == "2.9.1", torch.__version__
assert numpy.__version__ == "2.2.6", numpy.__version__

if args.flavor == "cu126":
    assert torch.version.cuda == "12.6", torch.version.cuda
    assert torch.distributed.is_nccl_available(), "NCCL ausente"

if args.gpu:
    assert torch.cuda.is_available(), "CUDA indisponível no container em execução"
    x = torch.randn(512, 512, device="cuda")
    y = x @ x
    torch.cuda.synchronize()

print(
    json.dumps(
        {
            "python": sys.version.split()[0],
            "executable": sys.executable,
            "pyspark": pyspark.__version__,
            "numpy": numpy.__version__,
            "torch": torch.__version__,
            "cuda_build": torch.version.cuda,
            "cuda_available": torch.cuda.is_available(),
            "gpu": torch.cuda.get_device_name(0) if args.gpu else None,
        },
        indent=2,
    )
)
