# Python SDK for ScopeConfig Service

This SDK provides a Python client for interacting with the ScopeConfig gRPC service.

## 1. Prerequisites

To use this SDK, you'll need to have Python 3 installed, along with the `grpcio` and `grpcio-tools` packages. It is recommended to use a virtual environment.

```bash
# Create and activate a virtual environment
python3 -m venv .venv
source .venv/bin/activate

# Install dependencies
pip install grpcio grpcio-tools
```

## 2. Generating the gRPC Client

This project uses a "code-generation" approach. The Python client code is generated from the `.proto` definition file and is **not** committed to the repository.

Before you can use the client, you must generate the gRPC code. Run the following command from the root of the project:

```bash
python -m grpc_tools.protoc -I. --python_out=./sdks/python --grpc_python_out=./sdks/python proto/config/v1/config.proto
```

This will generate `config_pb2.py` and `config_pb2_grpc.py` inside the `sdks/python` directory. These files are required to run the client but should not be part of your commits.

## 3. Usage

Here's an example of how to use the `ScopeConfigClient` to get a configuration:

```python
from client import ScopeConfigClient, Scope

# Create a client
client = ScopeConfigClient("localhost:50051")

# Define the configuration identifier
service_name = "my-service"
scope = Scope.SERVICE

# Get the configuration
config = client.get_config(service_name, scope=scope)

if config:
    print(f"Successfully retrieved configuration for {service_name}")
    for field in config.fields:
        print(f"  {field.path}: {field.value}")

```
