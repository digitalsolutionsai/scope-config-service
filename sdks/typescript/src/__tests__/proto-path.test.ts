import { getProtoPath, resolveProtoPath } from "../client";
import * as path from "path";
import * as fs from "fs";

// Mock fs module
jest.mock("fs");
const mockFs = fs as jest.Mocked<typeof fs>;

// Mock the gRPC modules to avoid needing a real server
jest.mock("@grpc/grpc-js", () => ({
  credentials: {
    createInsecure: jest.fn(),
    createSsl: jest.fn(),
  },
  loadPackageDefinition: jest.fn(),
  status: {
    NOT_FOUND: 5,
    INVALID_ARGUMENT: 3,
    UNAVAILABLE: 14,
  },
}));
jest.mock("@grpc/proto-loader", () => ({
  loadSync: jest.fn(),
}));

describe("getProtoPath", () => {
  it("returns the package-relative proto path", () => {
    const result = getProtoPath();
    expect(result).toContain(
      path.join("proto", "config", "v1", "config.proto")
    );
    // Should be an absolute path
    expect(path.isAbsolute(result)).toBe(true);
  });
});

describe("resolveProtoPath", () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.replaceProperty(process, "env", { ...originalEnv });
    delete process.env.SCOPE_CONFIG_PROTO_PATH;
    mockFs.existsSync.mockReset();
  });

  afterEach(() => {
    jest.replaceProperty(process, "env", originalEnv);
  });

  it("returns explicit protoPath when file exists", () => {
    const customPath = "/custom/path/config.proto";
    mockFs.existsSync.mockReturnValue(true);

    const result = resolveProtoPath(customPath);
    expect(result).toBe(customPath);
    expect(mockFs.existsSync).toHaveBeenCalledWith(customPath);
  });

  it("throws when explicit protoPath file does not exist", () => {
    const customPath = "/nonexistent/config.proto";
    mockFs.existsSync.mockReturnValue(false);

    expect(() => resolveProtoPath(customPath)).toThrow(
      "Proto file not found at specified path"
    );
  });

  it("returns SCOPE_CONFIG_PROTO_PATH env var when file exists", () => {
    const envPath = "/env/path/config.proto";
    process.env.SCOPE_CONFIG_PROTO_PATH = envPath;
    mockFs.existsSync.mockReturnValue(true);

    const result = resolveProtoPath();
    expect(result).toBe(envPath);
  });

  it("falls back to package-relative path when env var file does not exist", () => {
    process.env.SCOPE_CONFIG_PROTO_PATH = "/nonexistent/env/path.proto";
    // First call: env var path → false, second call: package path → true
    mockFs.existsSync.mockImplementation((p: fs.PathLike) => {
      return (
        String(p) !== "/nonexistent/env/path.proto" &&
        String(p).includes(path.join("proto", "config", "v1", "config.proto"))
      );
    });

    const result = resolveProtoPath();
    expect(result).toContain(
      path.join("proto", "config", "v1", "config.proto")
    );
  });

  it("returns package-relative path when it exists", () => {
    mockFs.existsSync.mockImplementation((p: fs.PathLike) => {
      // Only the package-relative path exists
      return String(p).includes("src") || String(p).includes("dist");
    });

    // If the package-relative path exists, it should be returned
    mockFs.existsSync.mockReturnValueOnce(true);
    const result = resolveProtoPath();
    expect(result).toContain(
      path.join("proto", "config", "v1", "config.proto")
    );
  });

  it("returns cwd-relative path when package path does not exist", () => {
    const cwdPath = path.resolve(
      process.cwd(),
      "proto",
      "config",
      "v1",
      "config.proto"
    );
    mockFs.existsSync.mockImplementation((p: fs.PathLike) => {
      return String(p) === cwdPath;
    });

    const result = resolveProtoPath();
    expect(result).toBe(cwdPath);
  });

  it("throws with helpful message when proto file not found anywhere", () => {
    mockFs.existsSync.mockReturnValue(false);

    expect(() => resolveProtoPath()).toThrow("Proto file not found");
    expect(() => resolveProtoPath()).toThrow("npx scopeconfig-copy-proto");
    expect(() => resolveProtoPath()).toThrow("SCOPE_CONFIG_PROTO_PATH");
  });
});
