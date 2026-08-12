import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.hoisted(() => vi.fn());

vi.mock("@/lib/api", () => ({
  get: mockGet,
}));

import { clearPublicKeyCache, fetchPublicKey } from "./sm2";

type PublicKeyResponse = {
  code: number;
  message: string;
  data: { publicKey: string };
};

describe("SM2 公钥缓存", () => {
  beforeEach(() => {
    mockGet.mockReset();
    clearPublicKeyCache();
  });

  it("较早请求迟到时不会覆盖较新请求写入的公钥", async () => {
    let resolveOld!: (response: PublicKeyResponse) => void;
    let resolveNew!: (response: PublicKeyResponse) => void;

    mockGet
      .mockImplementationOnce(
        () =>
          new Promise<PublicKeyResponse>((resolve) => {
            resolveOld = resolve;
          })
      )
      .mockImplementationOnce(
        () =>
          new Promise<PublicKeyResponse>((resolve) => {
            resolveNew = resolve;
          })
      );

    const oldRequest = fetchPublicKey(true);
    clearPublicKeyCache();
    const newRequest = fetchPublicKey(true);

    resolveNew({
      code: 0,
      message: "success",
      data: { publicKey: "NEW_PUBLIC_KEY" },
    });
    await expect(newRequest).resolves.toBe("NEW_PUBLIC_KEY");

    resolveOld({
      code: 0,
      message: "success",
      data: { publicKey: "OLD_PUBLIC_KEY" },
    });
    await expect(oldRequest).resolves.toBe("OLD_PUBLIC_KEY");

    await expect(fetchPublicKey()).resolves.toBe("NEW_PUBLIC_KEY");
    expect(mockGet).toHaveBeenCalledTimes(2);
  });
});
