/**
 * gRPC client for the ScopeConfig service
 */

import * as grpc from "@grpc/grpc-js";
import * as protoLoader from "@grpc/proto-loader";
import * as path from "path";
import {
  ClientOptions,
  ConfigIdentifier,
  ScopeConfig,
  ConfigServiceError,
} from "./types";

/**
 * ScopeConfig gRPC Client
 *
 * @example
 * ```typescript
 * // Create a client
 * const client = new ConfigClient({
 *   address: 'localhost:50051',
 *   insecure: true, // Use credentials for production
 * });
 *
 * await client.connect();
 *
 * const config = await client.getConfig(identifier);
 *
 * const latestConfig = await client.getLatestConfig(identifier);
 *
 * // Close connection
 * await client.close();
 * ```
 */
export class ConfigClient {
  private client: any;
  private options: ClientOptions;

  constructor(options: ClientOptions) {
    this.options = options;
  }

  /**
   * Connects to the gRPC server
   */
  async connect(): Promise<void> {
    try {
      const PROTO_PATH = path.join(
        __dirname,
        "../../proto/config/v1/config.proto"
      );
      const packageDefinition = protoLoader.loadSync(PROTO_PATH, {
        keepCase: true,
        longs: String,
        enums: String,
        defaults: true,
        oneofs: true,
      });

      const protoDescriptor = grpc.loadPackageDefinition(
        packageDefinition
      ) as any;
      const configService = protoDescriptor.vn.dsai.config.v1;

      const credentials = this.options.insecure
        ? grpc.credentials.createInsecure()
        : this.options.credentials || grpc.credentials.createSsl();

      this.client = new configService.ConfigService(
        this.options.address,
        credentials,
        this.options.channelOptions
      );
    } catch (error) {
      throw new Error(`Failed to connect to ConfigService: ${error}`);
    }
  }

  /**
   * Closes the client connection
   */
  async close(): Promise<void> {
    if (this.client) {
      this.client.close();
    }
  }

  async getConfig(identifier: ConfigIdentifier): Promise<ScopeConfig> {
    return this.promisify("GetConfig", { identifier });
  }

  async getLatestConfig(identifier: ConfigIdentifier): Promise<ScopeConfig> {
    return this.promisify("GetLatestConfig", { identifier });
  }

  /**
   * Promisifies a gRPC call and handles errors
   */
  private promisify(method: string, request: any): Promise<any> {
    return new Promise((resolve, reject) => {
      this.client[method](
        request,
        (error: grpc.ServiceError | null, response: any) => {
          if (error) {
            reject(this.wrapError(method, error));
          } else {
            resolve(response);
          }
        }
      );
    });
  }

  /**
   * Wraps gRPC errors with additional context
   */
  private wrapError(
    method: string,
    error: grpc.ServiceError
  ): ConfigServiceError {
    const statusCode = error.code;
    const message = error.details || error.message;

    switch (statusCode) {
      case grpc.status.NOT_FOUND:
        return new ConfigServiceError(
          `${method}: resource not found: ${message}`,
          statusCode,
          message
        );
      case grpc.status.INVALID_ARGUMENT:
        return new ConfigServiceError(
          `${method}: invalid argument: ${message}`,
          statusCode,
          message
        );
      case grpc.status.UNAVAILABLE:
        return new ConfigServiceError(
          `${method}: service unavailable: ${message}`,
          statusCode,
          message
        );
      default:
        return new ConfigServiceError(
          `${method} failed: ${message}`,
          statusCode,
          message
        );
    }
  }
}
