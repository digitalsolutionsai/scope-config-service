import grpc
from . import config_pb2
from . import config_pb2_grpc

class Scope:
    DEFAULT = config_pb2.DEFAULT
    SYSTEM = config_pb2.SYSTEM
    SERVICE = config_pb2.SERVICE
    PROJECT = config_pb2.PROJECT
    STORE = config_pb2.STORE

class ScopeConfigClient:
    def __init__(self, address):
        self.channel = grpc.insecure_channel(address)
        self.stub = config_pb2_grpc.ConfigServiceStub(self.channel)

    def get_config(self, service_name, project_id=None, store_id=None, group_id=None, scope=Scope.DEFAULT):
        request = config_pb2.GetConfigRequest(
            identifier=config_pb2.ConfigIdentifier(
                service_name=service_name,
                project_id=project_id,
                store_id=store_id,
                group_id=group_id,
                scope=scope,
            )
        )
        try:
            return self.stub.GetConfig(request)
        except grpc.RpcError as e:
            print(f"Error getting config: {e.details()}")
            return None
