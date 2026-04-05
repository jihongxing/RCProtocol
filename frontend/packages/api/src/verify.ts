export type VerificationResult = {
  status: "verified" | "restricted" | "failed";
  assetId?: string;
  message: string;
};

export async function verifyAsset(): Promise<VerificationResult> {
  return {
    status: "failed",
    message: "verify sdk placeholder",
  };
}
