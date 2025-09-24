import unittest
import grpc
import os

# Assuming the generated protobuf files are in a 'gen' directory
# You may need to adjust the Python path to include the 'sdks/python' and 'sdks/python/gen' directories
# export PYTHONPATH=$PYTHONPATH:./sdks/python:./sdks/python/gen
from config.v1 import config_service_pb2_grpc, config_service_pb2

class TestTemplateService(unittest.TestCase):

    def setUp(self):
        """Set up a gRPC channel and client stub for each test."""
        self.channel = grpc.insecure_channel('localhost:50051')
        self.stub = config_service_pb2_grpc.ConfigServiceStub(self.channel)
        self.user = "python-test-runner"

    def tearDown(self):
        """Close the channel after each test."""
        self.channel.close()

    def test_apply_and_get_template(self):
        """
        Test applying a configuration template and then retrieving it
        to ensure it was saved correctly.
        """
        service_name = "test-service-python"
        group_id = "test-group-python"

        # 1. Apply a Config Template
        template_to_apply = config_service_pb2.ConfigTemplate(
            identifier=config_service_pb2.ConfigIdentifier(
                service_name=service_name,
                group_id=group_id
            ),
            service_label="Test Service from Python",
            group_label="Test Group from Python",
            group_description="A test group created during automated Python tests.",
            fields=[
                config_service_pb2.ConfigFieldTemplate(
                    path="log.level",
                    label="Logging Level",
                    description="Controls the verbosity of application logging.",
                    type=config_service_pb2.FieldType.Value("STRING"),
                    default_value="INFO",
                    options=[
                        config_service_pb2.ValueOption(value="DEBUG", label="Debug"),
                        config_service_pb2.ValueOption(value="INFO", label="Info"),
                        config_service_pb2.ValueOption(value="WARN", label="Warning"),
                        config_service_pb2.ValueOption(value="ERROR", label="Error"),
                    ]
                )
            ]
        )

        apply_request = config_service_pb2.ApplyConfigTemplateRequest(
            template=template_to_apply,
            user=self.user
        )

        try:
            applied_template = self.stub.ApplyConfigTemplate(apply_request)
            print(f"Successfully applied template for {service_name}/{group_id}")
        except grpc.RpcError as e:
            self.fail(f"ApplyConfigTemplate failed with status {e.code()}: {e.details()}")

        # 2. Get the Config Template to verify
        get_request = config_service_pb2.GetConfigTemplateRequest(
            identifier=config_service_pb2.ConfigIdentifier(
                service_name=service_name,
                group_id=group_id
            )
        )

        try:
            retrieved_template = self.stub.GetConfigTemplate(get_request)
            print(f"Successfully retrieved template for {service_name}/{group_id}")
        except grpc.RpcError as e:
            self.fail(f"GetConfigTemplate failed with status {e.code()}: {e.details()}")

        # 3. Assert that the retrieved data matches the applied data
        self.assertEqual(retrieved_template.identifier.service_name, service_name)
        self.assertEqual(retrieved_template.identifier.group_id, group_id)
        self.assertEqual(retrieved_template.group_label, "Test Group from Python")
        self.assertEqual(len(retrieved_template.fields), 1)
        self.assertEqual(retrieved_template.fields[0].path, "log.level")
        self.assertEqual(len(retrieved_template.fields[0].options), 4)
        self.assertEqual(retrieved_template.fields[0].options[0].value, "DEBUG")


if __name__ == '__main__':
    unittest.main()
