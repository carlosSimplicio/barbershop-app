export async function register() {
  if (process.env.NEXT_RUNTIME === "nodejs") {
    const { getServerConfig } =
      await import("@/lib/establishment/public-establishment-server");
    getServerConfig();
  }
}
