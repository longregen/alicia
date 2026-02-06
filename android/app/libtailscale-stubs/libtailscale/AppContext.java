package libtailscale;

public interface AppContext {
    void log(String tag, String logLine);
    void encryptToPref(String key, String value) throws Exception;
    String decryptFromPref(String key) throws Exception;
    String getStateStoreKeysJSON();
    String getOSVersion() throws Exception;
    String getDeviceName() throws Exception;
    String getInstallSource();
    boolean shouldUseGoogleDNSFallback();
    boolean isChromeOS() throws Exception;
    String getInterfacesAsJson() throws Exception;
    String getPlatformDNSConfig();
    String getSyspolicyStringValue(String key) throws Exception;
    boolean getSyspolicyBooleanValue(String key) throws Exception;
    String getSyspolicyStringArrayJSONValue(String key) throws Exception;
    boolean hardwareAttestationKeySupported();
    String hardwareAttestationKeyCreate() throws Exception;
    void hardwareAttestationKeyRelease(String id) throws Exception;
    byte[] hardwareAttestationKeyPublic(String id) throws Exception;
    byte[] hardwareAttestationKeySign(String id, byte[] data) throws Exception;
    void hardwareAttestationKeyLoad(String id) throws Exception;
}
