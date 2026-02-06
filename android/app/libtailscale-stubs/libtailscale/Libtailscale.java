package libtailscale;

/**
 * Stub for gomobile-generated libtailscale bindings.
 * Replaced at runtime by the real libtailscale.aar when present.
 */
public class Libtailscale {
    public static Application start(String dataDir, String directFileRoot, boolean hwAttestation, AppContext appCtx) {
        throw new UnsupportedOperationException("libtailscale.aar not available");
    }

    public static void requestVPN(IPNService service) {
        throw new UnsupportedOperationException("libtailscale.aar not available");
    }

    public static void serviceDisconnect(IPNService service) {
        throw new UnsupportedOperationException("libtailscale.aar not available");
    }

    public static void onDNSConfigChanged(String ifname) {}

    public static void sendLog(byte[] logStr) {}
}
